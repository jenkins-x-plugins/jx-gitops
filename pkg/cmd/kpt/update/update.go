package update

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/jenkins-x/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"

	"github.com/jenkins-x/jx-helpers/v3/pkg/files"

	"github.com/jenkins-x/jx-gitops/pkg/plugins"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/v3/pkg/maps"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	kptLong = templates.LongDesc(`
		Updates any kpt packages installed in a sub directory

		If you know a specific directory which needs updating you can always use 'kpt' directly via:

  		    kpt pkg update mySubDir
`)

	kptExample = templates.Examples(`
		# recurses the current dir looking for directories with Kptfile inside 
		# and upgrades the kpt package found there to the latest version
		%s kpt --dir .
	`)

	pathSeparator = string(os.PathSeparator)

	info = termcolor.ColorInfo
)

// KptOptions the options for the command
type Options struct {
	Dir                    string
	Version                string
	RepositoryURL          string
	RepositoryOwner        string
	RepositoryName         string
	KptBinary              string
	Strategy               string
	IgnoreYamlContentError bool
	CommandRunner          cmdrunner.CommandRunner
}

// NewCmdKptUpdate creates a command object for the command
func NewCmdKptUpdate() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "update",
		Short:   "Updates any kpt packages installed in a sub directory",
		Long:    kptLong,
		Example: fmt.Sprintf(kptExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.AddFlags(cmd)
	return cmd, o
}

// AddFlags adds CLI flags
func (o *Options) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.Dir, "dir", "", ".", "the directory to recursively look for the *.yaml or *.yml files")
	cmd.Flags().StringVarP(&o.Version, "version", "v", "", "the git version of the kpt package to upgrade to")
	cmd.Flags().StringVarP(&o.RepositoryURL, "url", "u", "", "filter on the Kptfile repository URL for which packages to update")
	cmd.Flags().StringVarP(&o.RepositoryOwner, "owner", "o", "", "filter on the Kptfile repository owner (user/organisation) for which packages to update")
	cmd.Flags().StringVarP(&o.RepositoryName, "repo", "r", "", "filter on the Kptfile repository name  for which packages to update")
	cmd.Flags().StringVarP(&o.KptBinary, "bin", "", "", "the 'kpt' binary name to use. If not specified this command will download the jx binary plugin into ~/.jx3/plugins/bin and use that")
	cmd.Flags().StringVarP(&o.Strategy, "strategy", "s", "alpha-git-patch", "the 'kpt' strategy to use. To see available strategies type 'kpt pkg update --help'. Typical values are: resource-merge, fast-forward, alpha-git-patch, force-delete-replace")

	cmd.Flags().BoolVarP(&o.IgnoreYamlContentError, "ignore-yaml-error", "", false, "ignore kpt errors of the form: yaml: did not find expected node content")
}

// Run implements the command
func (o *Options) Run() error {
	if o.Dir == "" {
		o.Dir = "."
	}
	dir, err := filepath.Abs(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to find abs dir of %s", o.Dir)
	}

	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
	}

	strategies, err := o.LoadOverrideStrategies()
	if err != nil {
		return errors.Wrap(err, "failed to load kpt merge override strategies")
	}

	bin := o.KptBinary
	if bin == "" {
		bin, err = plugins.GetKptBinary(plugins.KptVersion)
		if err != nil {
			return err
		}
	}

	homedir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		kptDir, name := filepath.Split(path)
		if name != "Kptfile" {
			return nil
		}
		flag, err := o.Matches(path)
		if err != nil {
			return errors.Wrapf(err, "failed to check if path matches %s", path)
		}
		if !flag {
			return nil
		}
		rel, err := filepath.Rel(dir, kptDir)
		if err != nil {
			return errors.Wrapf(err, "failed to calculate the relative directory of %s", kptDir)
		}
		kptDir = strings.TrimSuffix(kptDir, pathSeparator)
		parentDir, _ := filepath.Split(kptDir)
		parentDir = strings.TrimSuffix(parentDir, pathSeparator)

		// clear the kpt repo cache everytime else we run into issues
		err = os.RemoveAll(filepath.Join(homedir, ".kpt", "repos"))
		if err != nil {
			return err
		}

		strategy := o.Strategy
		log.Logger().Infof("looking at dir %s in %v", rel, strategies)
		if strategies[rel] != "" {
			strategy = strategies[rel]
		}

		folderExpression := rel
		if o.Version != "" {
			folderExpression = fmt.Sprintf("%s@%s", rel, o.Version)
		} else {
			node, err := yaml.ReadFile(path)
			if err == nil {
				refNode, err := node.Pipe(yaml.Lookup("upstream", "git", "ref"))
				if err == nil {
					nodeText, err := refNode.String()
					if err != nil {
						folderExpression = fmt.Sprintf("%s@%s", rel, strings.TrimSpace(nodeText))
					}
				}
			}
		}

		args := []string{"pkg", "update", folderExpression, "--strategy", strategy}
		c := &cmdrunner.Command{
			Name: bin,
			Args: args,
			Dir:  dir,
		}
		text, err := o.CommandRunner(c)
		if err != nil {
			lines := strings.Split(strings.TrimSpace(text), "\n")
			errText := strings.ToLower(lines[len(lines)-1])
			if errText == "error: no updates" {
				return nil
			}

			if o.IgnoreYamlContentError && strings.Contains(text, "yaml: did not find expected node content") {
				return nil
			}
			if strings.Contains(text, "update failed") {
				handled, err2 := o.handleKptfileConflictsAndContinue(dir, lines)
				if handled && err2 == nil {
					return nil
				}
			}
			return errors.Wrapf(err, "failed to run kpt command")
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to upgrade kpt packages in dir %s", dir)
	}
	return nil
}

// handleKptfileConflictsAndContinue if there's only a single conflict for the Kptfile lets
// handle it and continue
func (o *Options) handleKptfileConflictsAndContinue(dir string, lines []string) (bool, error) {
	conflicts := 0
	kptConflict := false

	for _, line := range lines {
		if strings.HasPrefix(line, "CONFLICT (content): Merge conflict in ") {
			conflicts++
			if strings.HasSuffix(line, "Kptfile") {
				kptConflict = true
			}
		}
	}
	if !kptConflict || conflicts != 1 {
		return false, nil
	}

	log.Logger().Infof("lets work around the Kptfile merge conflict that kpt generated - its probably whitespace related...")

	// lets accept their change to the Kptfile and continue with the merge
	argsList := [][]string{
		{"checkout", "--theirs", "."},
		{"add", "-u"},
		{"am", "--continue"},
	}
	for _, args := range argsList {
		c := &cmdrunner.Command{
			Dir:  dir,
			Name: "git",
			Args: args,
		}
		_, err := o.CommandRunner(c)
		if err != nil {
			return false, errors.Wrapf(err, "failed to run %s", c.CLI())
		}
	}
	return true, nil
}

// Matches returns true if this kpt file matches the filters
func (o *Options) Matches(path string) (bool, error) {
	if o.RepositoryName == "" && o.RepositoryOwner == "" && o.RepositoryURL == "" {
		return true, nil
	}

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return false, errors.Wrapf(err, "failed to load file %s", path)
	}

	obj := &unstructured.Unstructured{}
	err = yaml.Unmarshal(data, obj)
	if err != nil {
		return false, errors.Wrapf(err, "failed to unmarshal YAML in file %s", path)
	}

	repoPath := "upstream.git.repo"
	repo := maps.GetMapValueAsStringViaPath(obj.Object, repoPath)
	if repo == "" {
		log.Logger().Warnf("could not find field %s in file %s", repoPath, path)
		return false, nil
	}
	if o.RepositoryURL != "" {
		if repo != o.RepositoryURL {
			return false, nil
		}
	}
	if o.RepositoryOwner != "" || o.RepositoryName != "" {
		gitInfo, err := giturl.ParseGitURL(repo)
		if err != nil {
			return false, errors.Wrapf(err, "failed to parse git URL %s", repo)
		}
		if o.RepositoryOwner != "" && o.RepositoryOwner != gitInfo.Organisation {
			return false, nil
		}
		if o.RepositoryName != "" && o.RepositoryName != gitInfo.Name {
			return false, nil
		}
	}
	return true, nil
}

func (o *Options) LoadOverrideStrategies() (map[string]string, error) {
	strategies := map[string]string{"versionStream": "force-delete-replace"}
	kptStrategyFilename := filepath.Join(o.Dir, ".jx", "gitops", v1alpha1.KptStragegyFileName)

	exists, err := files.FileExists(kptStrategyFilename)
	if !exists {
		log.Logger().Infof("no local strategy file %s found so using default merge strategies", info(kptStrategyFilename))
		return strategies, nil
	}
	data, err := ioutil.ReadFile(kptStrategyFilename)
	if err != nil {
		return strategies, errors.Wrapf(err, "failed to read kpt strategy file %s", kptStrategyFilename)
	}
	kptStrategies := &v1alpha1.KptStrategies{}
	err = yaml.Unmarshal(data, kptStrategies)
	if err != nil {
		return strategies, errors.Wrapf(err, "failed to unmarshall kpt strategy file %s", kptStrategyFilename)
	}
	err = kptStrategies.Validate()
	if err != nil {
		return strategies, errors.Wrapf(err, "failed to validate kpt strategy file %s", kptStrategyFilename)
	}
	for _, fileStrategy := range kptStrategies.KptStrategyConfig {
		strategies[fileStrategy.RelativePath] = fileStrategy.Strategy
	}
	return strategies, nil
}
