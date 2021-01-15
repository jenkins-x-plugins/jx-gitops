package report

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/helmfiles"
	"github.com/jenkins-x/jx-gitops/pkg/plugins"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/helmer"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yaml2s"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/roboll/helmfile/pkg/state"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/chart"
	"sigs.k8s.io/yaml"
)

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Generates a markdown report of the helmfile based deployments in each namespace
`)

	cmdExample = templates.Examples(`
		# generates a report of the deployments
		%s helmfile report
	`)

	valueFileNames = []string{"values.yaml.gotmpl", "values.yaml"}
	pathSeparator  = string(os.PathSeparator)
)

// Options the options for the command
type Options struct {
	options.BaseOptions
	Dir              string
	OutDir           string
	Namespace        string
	GitCommitMessage string
	Helmfile         string
	Helmfiles        []helmfiles.Helmfile
	HelmBinary       string
	DoGitCommit      bool
	Gitter           gitclient.Interface
	CommandRunner    cmdrunner.CommandRunner
	HelmClient       helmer.Helmer
	NamespaceCharts  []*NamespaceCharts
}

// NewCmdHelmfileReport creates a command object for the command
func NewCmdHelmfileReport() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "report",
		Short:   "Generates a markdown report of the helmfile based deployments in each namespace",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.HelmBinary, "helm-binary", "", "", "specifies the helm binary location to use. If not specified defaults to using the downloaded helm plugin")
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory that contains the helmfile.yaml")
	cmd.Flags().StringVarP(&o.OutDir, "out-dir", "o", "docs", "the output directory")
	o.AddFlags(cmd, "")
	o.BaseOptions.AddBaseFlags(cmd)
	return cmd, o
}

func (o *Options) AddFlags(cmd *cobra.Command, prefix string) {
	cmd.Flags().StringVarP(&o.Helmfile, "helmfile", "", "", "the helmfile to resolve. If not specified defaults to 'helmfile.yaml' in the dir")
	cmd.Flags().StringVarP(&o.GitCommitMessage, prefix+"commit-message", "", "chore: generated kubernetes resources from helm chart", "the git commit message used")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", "jx", "the default namespace if none is specified in the helmfile.yaml")

	// git commit stuff....
	cmd.Flags().BoolVarP(&o.DoGitCommit, prefix+"git-commit", "", false, "if set then the template command will git commit the modified helmfile.yaml files")
}

// Validate validates the options and populates any missing values
func (o *Options) Validate() error {
	if o.Helmfile == "" {
		o.Helmfile = filepath.Join(o.Dir, "helmfile.yaml")
	}

	var err error
	o.Helmfiles, err = helmfiles.GatherHelmfiles(o.Helmfile)
	if err != nil {
		return errors.Wrapf(err, "failed to gather nested helmfiles")
	}

	if o.GitCommitMessage == "" {
		o.GitCommitMessage = "chore: resolved charts and values from the version stream"
	}
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.QuietCommandRunner
	}

	if o.HelmBinary == "" {
		o.HelmBinary, err = plugins.GetHelmBinary(plugins.HelmVersion)
		if err != nil {
			return errors.Wrapf(err, "failed to download helm plugin")
		}
	}
	if o.HelmClient == nil {
		o.HelmClient = helmer.NewHelmCLIWithRunner(o.CommandRunner, o.HelmBinary, "", false)
	}
	err = os.MkdirAll(o.OutDir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create output dir %s", o.OutDir)
	}
	return nil
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to ")
	}

	for _, hf := range o.Helmfiles {
		charts, err := o.processHelmfile(hf)
		if err != nil {
			return errors.Wrapf(err, "failed to process helmfile %s", hf.Filepath)
		}
		if charts != nil {
			o.NamespaceCharts = append(o.NamespaceCharts, charts)
		}
	}

	path := filepath.Join(o.OutDir, "releases.yaml")
	err = yamls.SaveFile(o.NamespaceCharts, path)
	if err != nil {
		return errors.Wrapf(err, "failed to save %s", path)
	}
	log.Logger().Infof("saved %s", info(path))

	md, err := ToMarkdown(o.NamespaceCharts)
	path = filepath.Join(o.OutDir, "README.md")
	err = ioutil.WriteFile(path, []byte(md), files.DefaultFileWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save %s", path)
	}
	log.Logger().Infof("saved %s", info(path))
	return nil
}

func (o *Options) processHelmfile(helmfile helmfiles.Helmfile) (*NamespaceCharts, error) {
	answer := &NamespaceCharts{}
	// ignore the root file
	if helmfile.RelativePathToRoot == "" {
		return nil, nil
	}
	helmState := &state.HelmState{}
	path := helmfile.Filepath
	err := yaml2s.LoadFile(path, helmState)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load helmfile %s", helmfile)
	}

	ns := helmState.OverrideNamespace
	if ns == "" {
		names := strings.Split(helmfile.Filepath, string(os.PathSeparator))
		if len(names) > 1 {
			ns = names[len(names)-2]
		}
	}
	answer.Namespace = ns
	answer.Path = helmfile.Filepath

	log.Logger().Infof("namespace %s", ns)

	for i := range helmState.Releases {
		rel := &helmState.Releases[i]
		ci, err := o.createReleaseInfo(helmState, rel)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create release info for %s", rel.Chart)
		}

		if info == nil {
			continue
		}
		answer.Charts = append(answer.Charts, ci)
		log.Logger().Infof("found %s", ci.String())
	}
	log.Logger().Infof("")
	return answer, nil
}

func (o *Options) createReleaseInfo(helmState *state.HelmState, rel *state.ReleaseSpec) (*ChartInfo, error) {
	chart := rel.Chart
	if chart == "" {
		return nil, nil
	}
	paths := strings.SplitN(chart, "/", 2)
	info := &ChartInfo{}
	info.Version = rel.Version
	switch len(paths) {
	case 0:
		return nil, nil
	case 1:
		info.Name = paths[0]
	default:
		info.RepositoryName = paths[0]
		info.Name = paths[1]
	}

	if info.RepositoryName != "" {
		// lets find the repo URL
		for i := range helmState.Repositories {
			repo := &helmState.Repositories[i]
			if repo.Name == info.RepositoryName {
				err := o.enrichChartMetadata(info, repo)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get chart metadata for %s", info.String())
				}
			}
		}
	}
	return info, nil
}

func (o *Options) enrichChartMetadata(i *ChartInfo, repo *state.RepositorySpec) error {
	version := i.Version
	name := i.Name
	repoURL := repo.URL
	if repoURL != "" {
		repoName := repo.Name
		if i.RepositoryURL == "" {
			i.RepositoryURL = repoURL
		}
		if i.RepositoryName == "" {
			i.RepositoryName = repoName
		}
		_, err := helmer.AddHelmRepoIfMissing(o.HelmClient, repoURL, repoName, repo.Username, repo.Password)
		if err != nil {
			return errors.Wrapf(err, "failed to add helm repository %s %s", repoName, repoURL)
		}
		log.Logger().Debugf("added helm repository %s %s", repoName, repoURL)
	}

	args := []string{"show", "chart"}
	if version != "" {
		args = append(args, "--version", version)
	}
	if repoURL != "" {
		args = append(args, "--repo", repoURL)
	}
	args = append(args, name)

	c := &cmdrunner.Command{
		Name: o.HelmBinary,
		Args: args,
	}
	text, err := o.CommandRunner(c)
	if err != nil {
		log.Logger().Warnf("failed to run %s", c.CLI())
		return nil
	}
	if strings.TrimSpace(text) == "" {
		log.Logger().Warnf("no output for %s", c.CLI())
		return nil
	}

	m := &chart.Metadata{}
	err = yaml.UnmarshalStrict([]byte(text), &m)
	if err != nil {
		return errors.Wrapf(err, "failed to parse the output of %s got results: %s", c.CLI(), text)
	}
	i.Metadata = *m
	i.Name = name
	i.Version = version
	return nil
}
