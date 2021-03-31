package report

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/helmfiles"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/plugins"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/releasereport"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/helmer"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/services"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yaml2s"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/roboll/helmfile/pkg/state"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/chart"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
	Dir                     string
	OutDir                  string
	ConfigRootPath          string
	Namespace               string
	GitCommitMessage        string
	Helmfile                string
	Helmfiles               []helmfiles.Helmfile
	HelmBinary              string
	DoGitCommit             bool
	Gitter                  gitclient.Interface
	CommandRunner           cmdrunner.CommandRunner
	HelmClient              helmer.Helmer
	Requirements            *jxcore.Requirements
	NamespaceCharts         []*releasereport.NamespaceReleases
	PreviousNamespaceCharts []*releasereport.NamespaceReleases
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
	cmd.Flags().StringVarP(&o.ConfigRootPath, "config-root", "", "config-root", "the folder name containing the kubernetes resources")
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
	o.Helmfiles, err = helmfiles.GatherHelmfiles(o.Helmfile, o.Dir)
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

	o.Requirements, _, err = jxcore.LoadRequirementsConfig(o.Dir, false)
	if err != nil {
		return errors.Wrapf(err, "failed to load requirements in dir %s", o.Dir)
	}
	return nil
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to ")
	}

	path := filepath.Join(o.OutDir, "releases.yaml")

	exists, err := files.FileExists(path)
	if err != nil {
		return errors.Wrapf(err, "failed to check file exists %s", path)
	}
	if exists {
		err = releasereport.LoadReleases(path, &o.PreviousNamespaceCharts)
		if err != nil {
			return err
		}
	}

	for _, hf := range o.Helmfiles {
		charts, err := o.processHelmfile(hf)
		if err != nil {
			return errors.Wrapf(err, "failed to process helmfile %s", hf.Filepath)
		}
		if charts != nil {
			for i, nc := range o.NamespaceCharts {
				// lets remove the old entry for the namespace
				if nc.Namespace == charts.Namespace {
					s := o.NamespaceCharts[0:i]
					if i+1 < len(o.NamespaceCharts) {
						s = append(s, o.NamespaceCharts[i+1:]...)
					}
					o.NamespaceCharts = s
				}
			}
			o.NamespaceCharts = append(o.NamespaceCharts, charts)
		}
	}

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

func (o *Options) processHelmfile(helmfile helmfiles.Helmfile) (*releasereport.NamespaceReleases, error) {
	answer := &releasereport.NamespaceReleases{}
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
		ci, err := o.createReleaseInfo(helmState, ns, rel)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create release info for %s", rel.Chart)
		}

		if info == nil {
			continue
		}
		answer.Releases = append(answer.Releases, ci)
		log.Logger().Infof("found %s", ci.String())
	}
	log.Logger().Infof("")
	return answer, nil
}

func (o *Options) createReleaseInfo(helmState *state.HelmState, ns string, rel *state.ReleaseSpec) (*releasereport.ReleaseInfo, error) {
	chart := rel.Chart
	if chart == "" {
		return nil, nil
	}
	paths := strings.SplitN(chart, "/", 2)
	answer := &releasereport.ReleaseInfo{}
	answer.Version = rel.Version
	switch len(paths) {
	case 0:
		return nil, nil
	case 1:
		answer.Name = paths[0]
	default:
		answer.RepositoryName = paths[0]
		answer.Name = paths[1]
	}

	if answer.RepositoryName != "" {
		// lets find the repo URL
		for i := range helmState.Repositories {
			repo := &helmState.Repositories[i]
			if repo.Name == answer.RepositoryName {
				err := o.enrichChartMetadata(answer, repo, rel, ns)
				if err != nil {
					return nil, errors.Wrapf(err, "failed to get chart metadata for %s", answer.String())
				}
				break
			}
		}
	}
	err := o.discoverResources(answer, ns, rel)
	if err != nil {
		return answer, errors.Wrapf(err, "failed to discover resources for %s", answer.String())
	}

	answer.LogsURL = getLogURL(&o.Requirements.Spec, ns, answer.Name)
	return answer, nil
}

func localName(chartName string) string {
	paths := strings.SplitN(chartName, "/", 2)
	if len(paths) == 2 {
		return paths[1]
	}
	return chartName
}

func (o *Options) enrichChartMetadata(i *releasereport.ReleaseInfo, repo *state.RepositorySpec, rel *state.ReleaseSpec, ns string) error {
	if repo.OCI {
		return nil
	}
	// lets see if we can find the previous data in the previous release
	localChartName := localName(rel.Chart)

	for _, nc := range o.PreviousNamespaceCharts {
		if nc.Namespace != ns {
			continue
		}
		for _, ch := range nc.Releases {
			if ch.Name == localChartName && ch.Version == rel.Version {
				*i = *ch
				// lets clear the old ingress/app URLs
				i.ApplicationURL = ""
				i.Ingresses = nil
				return nil
			}
		}
	}

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

func (o *Options) discoverResources(ci *releasereport.ReleaseInfo, ns string, rel *state.ReleaseSpec) error {
	namespaceDir := filepath.Join(o.Dir, o.ConfigRootPath, "namespaces", ns)
	exists, err := files.DirExists(namespaceDir)
	if err != nil {
		return errors.Wrapf(err, "failed to check if dir exists %s", namespaceDir)
	}
	if !exists {
		return nil
	}

	// lets try find the resources folder
	names := []string{ci.Name + "-" + rel.Name, ci.Name}
	for _, name := range names {
		chartDir := filepath.Join(namespaceDir, name)
		exists, err := files.DirExists(chartDir)
		if err != nil {
			return errors.Wrapf(err, "failed to check if dir exists %s", chartDir)
		}
		if exists {
			ci.ResourcesPath, err = filepath.Rel(o.Dir, chartDir)
			if err != nil {
				return errors.Wrapf(err, "failed to resolve relative path %s from %s", chartDir, o.Dir)
			}
			err = o.discoverIngress(ci, ns, rel, chartDir)
			if err != nil {
				return errors.Wrapf(err, "failed to discover ingress")
			}
			return nil
		}
	}
	return nil
}

func (o *Options) discoverIngress(ci *releasereport.ReleaseInfo, ns string, rel *state.ReleaseSpec, resourcesDir string) error {
	fs, err := ioutil.ReadDir(resourcesDir)
	if err != nil {
		return errors.Wrapf(err, "failed to read dir %s", resourcesDir)
	}

	for _, f := range fs {
		if f.IsDir() {
			continue
		}
		name := f.Name()
		if !strings.HasSuffix(name, ".yaml") && !strings.HasSuffix(name, ".yml") {
			continue
		}

		path := filepath.Join(resourcesDir, name)
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to load file %s", path)
		}

		obj := &unstructured.Unstructured{}
		err = yaml.Unmarshal(data, obj)
		if err != nil {
			log.Logger().Infof("could not parse YAML file %s", path)
			continue
		}
		if obj.GetKind() != "Ingress" {
			continue
		}
		apiVersion := obj.GetAPIVersion()
		if apiVersion != "networking.k8s.io/v1beta1" && apiVersion != "extensions/v1beta1" {
			log.Logger().Infof("ignoring Ingress in file %s with api version %s", path, apiVersion)
			continue
		}

		ing := &v1beta1.Ingress{}
		err = yaml.Unmarshal(data, ing)
		if err != nil {
			log.Logger().Warnf("failed to unmarshal YAML as Ingress in file %s: %s", path, err.Error())
			continue
		}

		u := services.IngressURL(ing)
		if u == "" {
			continue
		}

		ci.Ingresses = append(ci.Ingresses, releasereport.IngressInfo{
			Name: ing.Name,
			URL:  u,
		})

		if ing.Name == ci.Name || ing.Name == ci.Name+"-"+rel.Name {
			ci.ApplicationURL = u
		}
	}
	return nil
}
