package report

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/helmfile/helmfile/pkg/state"
	charter "github.com/jenkins-x-plugins/jx-charter/pkg/apis/chart/v1alpha1"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/helmfiles"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/helmhelpers"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/plugins"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/releasereport"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
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
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/helmpath"
	helmrepo "helm.sh/helm/v3/pkg/repo"
	nv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	PreviousNamespaceCharts map[string]map[string]*releasereport.ReleaseInfo
	RepositoryInfo          map[string]*helmrepo.IndexFile
	HelmSettings            *cli.EnvSettings
}

// NewCmdHelmfileReport creates a command object for the command
func NewCmdHelmfileReport() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "report",
		Short:   "Generates a markdown report of the helmfile based deployments in each namespace",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(_ *cobra.Command, _ []string) {
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

	o.HelmSettings = cli.New()
	o.RepositoryInfo = make(map[string]*helmrepo.IndexFile)
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
		var previousNamespaceCharts []*releasereport.NamespaceReleases
		err = releasereport.LoadReleases(path, &previousNamespaceCharts)
		if err != nil {
			return err
		}
		chartMap := map[string]map[string]*releasereport.ReleaseInfo{}
		for _, nc := range previousNamespaceCharts {
			nsMap, found := chartMap[nc.Namespace]
			if !found {
				nsMap = map[string]*releasereport.ReleaseInfo{}
				chartMap[nc.Namespace] = nsMap
			}
			for _, ri := range nc.Releases {
				nsMap[ri.ReleaseName] = ri
			}
		}
		o.PreviousNamespaceCharts = chartMap
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
	if err != nil {
		return errors.Wrap(err, "failed to convert charts to markdown")
	}
	path = filepath.Join(o.OutDir, "README.md")
	err = os.WriteFile(path, []byte(md), files.DefaultFileWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save %s", path)
	}
	log.Logger().Infof("saved %s", info(path))

	return o.generateChartCRDs()
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
		exists, err := o.verifyReleaseExists(ns, rel)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to verify if release exists %s in namespace %s", rel.Chart, ns)
		}
		if !exists {
			if rel.Condition != "" {
				log.Logger().Infof("ignoring release %s in namespace %s as using conditional %s", info(rel.Chart), info(ns), info(rel.Condition))
				continue
			}
			if rel.Installed != nil && !*rel.Installed {
				log.Logger().Infof("ignoring release %s in namespace %s as it isn't installed", info(rel.Chart), info(ns))
				continue
			}
			log.Logger().Warnf("ignoring release %s in namespace %s as we cannot find any generated resources but there is no conditional", rel.Chart, ns)
			continue
		}

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
	chartName := rel.Chart
	if chartName == "" {
		return nil, nil
	}
	paths := strings.SplitN(chartName, "/", 2)
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

	if !helmhelpers.IsChartNameRelative(chartName) && !helmhelpers.IsChartRemote(chartName) {
		if answer.FirstDeployed == nil {
			answer.FirstDeployed = createNow()
		}
		if answer.LastDeployed.IsZero() {
			answer.LastDeployed = createNow()
		}
	}

	answer.ReleaseName = rel.Name
	answer.LogsURL = getLogURL(&o.Requirements.Spec, ns, answer.Name)
	return answer, nil
}

func (o *Options) enrichChartMetadata(i *releasereport.ReleaseInfo, repo *state.RepositorySpec, rel *state.ReleaseSpec, ns string) (err error) {
	if repo.OCI {
		return nil
	}
	// lets see if we can find the previous data in the previous release
	if nsMap, found := o.PreviousNamespaceCharts[ns]; found {
		ch := nsMap[rel.Name]
		if ch != nil {
			if ch.Version == rel.Version {
				*i = *ch
				// let's clear the old ingress/app URLs
				i.ApplicationURL = ""
				i.Ingresses = nil
				return nil
			}
			i.FirstDeployed = ch.FirstDeployed
		}
	}
	name := i.Name
	repoURL := repo.URL
	repoName := repo.Name

	i.Name = name
	i.RepositoryURL = repoURL
	i.RepositoryName = repoName
	// Is there any valid case where the repo URL is empty?
	if repoURL == "" {
		return nil
	}

	indexFile, exists := o.RepositoryInfo[i.RepositoryURL]
	if !exists {
		repoName, err = helmer.AddHelmRepoIfMissing(o.HelmClient, repoURL, repoName, repo.Username, repo.Password)
		if err != nil {
			return errors.Wrapf(err, "failed to add helm repository %s %s", repo.Name, repoURL)
		}
		log.Logger().Debugf("added helm repository %s %s", repo.Name, repoURL)
		path := filepath.Join(o.HelmSettings.RepositoryCache, helmpath.CacheIndexFile(repoName))

		indexFile, err = helmrepo.LoadIndexFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to open repository index file %s", path)
		}
		o.RepositoryInfo[i.RepositoryURL] = indexFile
	}
	chartVersion, err := indexFile.Get(name, i.Version)
	if err != nil {
		return errors.Wrapf(err, "failed to find chart %s in repository index file for %s", name, i.RepositoryURL)
	}
	i.Metadata = *chartVersion.Metadata
	i.Version = chartVersion.Version
	return nil
}

func createNow() *metav1.Time {
	return &metav1.Time{
		Time: time.Now(),
	}
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
			err = o.discoverIngress(ci, rel, chartDir)
			if err != nil {
				return errors.Wrapf(err, "failed to discover ingress")
			}
			return nil
		}
	}
	return nil
}

func (o *Options) discoverIngress(ci *releasereport.ReleaseInfo, rel *state.ReleaseSpec, resourcesDir string) error {
	fs, err := os.ReadDir(resourcesDir)
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
		data, err := os.ReadFile(path)
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
		if apiVersion != "networking.k8s.io/v1beta1" && apiVersion != "extensions/v1beta1" && apiVersion != "networking.k8s.io/v1" {
			log.Logger().Infof("ignoring Ingress in file %s with api version %s", path, apiVersion)
			continue
		}

		ing := &nv1.Ingress{}
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

func (o *Options) generateChartCRDs() error {
	// lets check if we have installed the jx-charter chart which if not we don't generate Chart CRDs
	// as we need the CRD to know if we should create them....
	found := false
	for _, nc := range o.NamespaceCharts {
		for _, r := range nc.Releases {
			if r.Name == "jx-charter" {
				found = true
				break
			}
		}
	}
	if !found {
		return nil
	}

	for _, nc := range o.NamespaceCharts {
		for _, r := range nc.Releases {
			ns := nc.Namespace
			name := r.Name

			// lets make sure we don't have a relative path or anything funky
			i := strings.LastIndex(name, "/")
			if i > 0 {
				name = name[i+1:]
			}
			if name == "" {
				continue
			}
			path := filepath.Join(o.Dir, o.ConfigRootPath, "namespaces", ns, "chart-crds", name+".yaml")
			dir := filepath.Dir(path)
			err := os.MkdirAll(dir, files.DefaultDirWritePermissions)
			if err != nil {
				return errors.Wrapf(err, "failed to create dir %s", dir)
			}
			status := &charter.ChartStatus{
				Description: "Install complete",
				Status:      "deployed",
				Notes:       "",
			}
			if r.FirstDeployed != nil {
				status.FirstDeployed = *r.FirstDeployed
			}
			if r.LastDeployed != nil {
				status.LastDeployed = *r.LastDeployed
			}
			ch := &charter.Chart{
				TypeMeta: metav1.TypeMeta{
					APIVersion: charter.APIVersion,
					Kind:       charter.KindChart,
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: ns,
				},
				Spec: charter.ChartSpec{
					Metadata:       r.Metadata,
					RepositoryName: r.RepositoryName,
					RepositoryURL:  r.RepositoryURL,
				},
				Status: status,
			}
			err = yamls.SaveFile(ch, path)
			if err != nil {
				return errors.Wrapf(err, "failed to save file %s", path)
			}
		}
	}
	return nil
}

func (o *Options) verifyReleaseExists(ns string, r *state.ReleaseSpec) (bool, error) {
	name := r.Name
	nsDir := filepath.Join(o.Dir, o.ConfigRootPath, "namespaces", ns)
	path := filepath.Join(nsDir, name)
	exists, err := files.DirExists(path)
	if err != nil {
		return false, errors.Wrapf(err, "failed to check if directory exist %s", path)
	}
	if exists {
		return true, nil
	}
	// lets see if we are using chartName-releaseName as the dir which `jx gitops move` uses
	localChartName := r.Chart
	i := strings.LastIndex(localChartName, "/")
	if i > 0 {
		localChartName = localChartName[i+1:]
	}
	releaseName := r.Name
	name = localChartName + "-" + releaseName
	path = filepath.Join(nsDir, name)
	exists, err = files.DirExists(path)
	if err != nil {
		return false, errors.Wrapf(err, "failed to check if directory exist %s", path)
	}
	return exists, nil
}
