package update

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/apis/gitops/v1alpha1"

	"github.com/jenkins-x/jx-helpers/pkg/files"

	"github.com/jenkins-x/jx-gitops/pkg/plugins"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/pkg/maps"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

var (
	kptLong = templates.LongDesc(`
		Updates the kpt packages in the given directory
`)

	kptExample = templates.Examples(`
		# updates the kpt of all the yaml resources in the given directory
		%s kpt --dir .
	`)

	pathSeparator = string(os.PathSeparator)
)

const (
	defaultKptStrategy = "alpha-git-patch"
)

// KptOptions the options for the command
type Options struct {
	Dir             string
	Version         string
	RepositoryURL   string
	RepositoryOwner string
	RepositoryName  string
	KptBinary       string
	CommandRunner   cmdrunner.CommandRunner
}

// NewCmdKptUpdate creates a command object for the command
func NewCmdKptUpdate() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "update",
		Short:   "Updates the kpt packages in the given directory",
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
	cmd.Flags().StringVarP(&o.Version, "version", "v", "master", "the git version of the kpt package to upgrade to")
	cmd.Flags().StringVarP(&o.RepositoryURL, "url", "u", "", "filter on the Kptfile repository URL for which packages to update")
	cmd.Flags().StringVarP(&o.RepositoryOwner, "owner", "o", "", "filter on the Kptfile repository owner (user/organisation) for which packages to update")
	cmd.Flags().StringVarP(&o.RepositoryName, "repo", "r", "", "filter on the Kptfile repository name  for which packages to update")
	cmd.Flags().StringVarP(&o.KptBinary, "bin", "", "", "the 'kpt' binary name to use. If not specified this command will download the jx binary plugin into ~/.jx3/plugins/bin and use that")
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

	usr, err := user.Current()
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
		err = os.RemoveAll(filepath.Join(usr.HomeDir, ".kpt", "repos"))
		if err != nil {
			return err
		}

		strategy := defaultKptStrategy
		log.Logger().Infof("looking at dir %s in %v", rel, strategies)
		if strategies[rel] != "" {
			strategy = strategies[rel]
		}

		folderExpression := fmt.Sprintf("%s@%s", rel, o.Version)
		args := []string{"pkg", "update", folderExpression, "--strategy", strategy}
		c := &cmdrunner.Command{
			Name: bin,
			Args: args,
			Dir:  dir,
		}
		log.Logger().Infof("about to run %s in dir %s", termcolor.ColorInfo(c.String()), termcolor.ColorInfo(c.Dir))
		text, err := o.CommandRunner(c)
		log.Logger().Infof(text)
		if err != nil {
			lines := strings.Split(strings.TrimSpace(text), "\n")
			errText := lines[len(lines)-1]
			if errText == "Error: no updates" {
				return nil
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
	strategies := map[string]string{}
	kptStrategyFilename := filepath.Join(o.Dir, ".jx", "gitops", v1alpha1.KptStragegyFileName)

	exists, err := files.FileExists(kptStrategyFilename)
	if !exists {
		log.Logger().Infof("no local strategy file %s found so using default merge strategies", kptStrategyFilename)
		return o.getDefaultOverrideStrategies(), nil
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

func (o *Options) getDefaultOverrideStrategies() map[string]string {
	return map[string]string{"versionStream": "force-delete-replace"}
}
