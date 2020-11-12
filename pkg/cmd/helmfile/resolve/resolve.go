package resolve

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-api/v3/pkg/config"
	"github.com/jenkins-x/jx-gitops/pkg/helmhelpers"
	"github.com/jenkins-x/jx-gitops/pkg/jxtmpl/reqvalues"
	"github.com/jenkins-x/jx-gitops/pkg/plugins"
	"github.com/jenkins-x/jx-gitops/pkg/versionstreamer"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yaml2s"
	"github.com/roboll/helmfile/pkg/state"

	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/versionstream"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Resolves the helmfile.yaml from the version stream to specify versions and helm values
`)

	cmdExample = templates.Examples(`
		# resolves the versions and values in the helmfile.yaml
		%s helmfile resolve
	`)

	valueFileNames = []string{"values.yaml.gotmpl", "values.yaml"}
)

// Options the options for the command
type Options struct {
	versionstreamer.Options
	Namespace        string
	GitCommitMessage string
	Helmfile         string
	KptBinary        string
	HelmBinary       string
	BatchMode        bool
	UpdateMode       bool
	DoGitCommit      bool
	TestOutOfCluster bool
	Gitter           gitclient.Interface
	prefixes         *versionstream.RepositoryPrefixes
	Results          Results
}

type Results struct {
	HelmState                  state.HelmState
	RequirementsValuesFileName string
}

// NewCmdHelmfileResolve creates a command object for the command
func NewCmdHelmfileResolve() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "resolve",
		Short:   "Resolves any missing versions or values files in the helmfile.yaml file from the version stream",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().BoolVarP(&o.UpdateMode, "update", "", false, "updates versions from the version stream if they have changed")
	cmd.Flags().StringVarP(&o.HelmBinary, "helm-binary", "", "", "specifies the helm binary location to use. If not specified defaults to using the downloaded helm plugin")
	o.AddFlags(cmd, "")
	return cmd, o
}

func (o *Options) AddFlags(cmd *cobra.Command, prefix string) {
	cmd.Flags().StringVarP(&o.Helmfile, "helmfile", "", "", "the helmfile to resolve. If not specified defaults to 'helmfile.yaml' in the dir")
	cmd.Flags().StringVarP(&o.GitCommitMessage, prefix+"commit-message", "", "chore: generated kubernetes resources from helm chart", "the git commit message used")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", "jx", "the default namespace if none is specified in the helmfile.yaml or jx-requirements.yml")

	// git commit stuff....
	cmd.Flags().BoolVarP(&o.DoGitCommit, prefix+"git-commit", "", false, "if set then the template command will git commit the modified helmfile.yaml files")
}

// Validate validates the options and populates any missing values
func (o *Options) Validate() error {
	err := o.Options.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to ")
	}

	if o.Helmfile == "" {
		o.Helmfile = filepath.Join(o.Dir, "helmfile.yaml")
	}

	err = yaml2s.LoadFile(o.Helmfile, &o.Results.HelmState)
	if err != nil {
		return errors.Wrapf(err, "failed to load helmfile %s", o.Helmfile)
	}

	if o.GitCommitMessage == "" {
		o.GitCommitMessage = "chore: resolved charts and values from the version stream"
	}

	o.prefixes, err = o.Options.Resolver.GetRepositoryPrefixes()
	if err != nil {
		return errors.Wrapf(err, "failed to load repository prefixes at %s", o.VersionStreamDir)
	}

	jxReqValuesFileName := filepath.Join(o.Dir, reqvalues.RequirementsValuesFileName)
	o.Results.RequirementsValuesFileName = reqvalues.RequirementsValuesFileName
	err = reqvalues.SaveRequirementsValuesFile(o.Options.Requirements, jxReqValuesFileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save tempo file for jx requirements values file %s", jxReqValuesFileName)
	}
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.QuietCommandRunner
	}
	return nil
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to ")
	}

	if o.UpdateMode {
		err = o.CustomUpgrades()
		if err != nil {
			return errors.Wrapf(err, "failed to perform custom upgrades")
		}
	}

	resolver := o.Options.Resolver
	if resolver == nil {
		return errors.Errorf("failed to create the VersionResolver")
	}
	versionsDir := resolver.VersionsDir
	appsCfgDir := o.Dir

	log.Logger().Infof("resolving versions and values files from the version stream %s ref %s in dir %s", o.VersionStreamURL, o.VersionStreamRef, o.VersionStreamDir)

	helmState := o.Results.HelmState

	var ignoreRepositories []string
	if !helmhelpers.IsInCluster() || o.TestOutOfCluster {
		ignoreRepositories, err = helmhelpers.FindClusterLocalRepositoryURLs(helmState.Repositories)
		if err != nil {
			return errors.Wrapf(err, "failed to find cluster local repositories")
		}
	}

	err = helmhelpers.AddHelmRepositories(o.HelmBinary, helmState, o.QuietCommandRunner, ignoreRepositories)
	if err != nil {
		return errors.Wrapf(err, "failed to add helm repositories")
	}

	/*
		TODO lazily create environments file?
		requirementsValuesFiles := o.Results.RequirementsValuesFileName
		if requirementsValuesFiles != "" {
			if stringhelpers.StringArrayIndex(helmState.Values, requirementsValuesFiles) < 0 {
				helmState.Values = append(helmState.Values, requirementsValuesFiles)
			}
		}

	*/
	count := 0
	for i, release := range helmState.Releases {
		// TODO
		//repository := release.Repository
		repository := ""
		fullChartName := release.Chart
		parts := strings.Split(fullChartName, "/")
		prefix := ""
		chartName := release.Chart
		if len(parts) > 1 {
			prefix = parts[0]
			chartName = parts[1]
		}
		if release.Name == "" {
			release.Name = chartName
		}

		// lets not try resolve repository / versions for local charts
		if prefix != "." && prefix != ".." {
			// lets resolve the chart prefix from a local repository from the file or from a
			// prefix in the versions stream
			if repository == "" && prefix != "" {
				for _, r := range helmState.Repositories {
					if r.Name == prefix {
						repository = r.URL
					}
				}
			}
			if repository == "" && prefix != "" {
				repository, err = versionstreamer.MatchRepositoryPrefix(o.prefixes, prefix)
				if err != nil {
					return errors.Wrapf(err, "failed to match prefix %s with repositories from versionstream %s", prefix, o.VersionStreamURL)
				}
			}
			if repository == "" && prefix != "" {
				return errors.Wrapf(err, "failed to find repository URL, not defined in helmfile.yaml or versionstream %s", o.VersionStreamURL)
			}
			if repository != "" && prefix != "" {
				// lets ensure we've got a repository for this URL in the apps file
				found := false
				for _, r := range helmState.Repositories {
					if r.Name == prefix {
						if r.URL != repository {
							return errors.Errorf("release %s has prefix %s for repository URL %s which is also mapped to prefix %s", release.Name, prefix, r.URL, r.Name)
						}
						found = true
						break
					}
				}
				if !found {
					helmState.Repositories = append(helmState.Repositories, state.RepositorySpec{
						Name: prefix,
						URL:  repository,
					})
				}
			}

			if stringhelpers.StringArrayIndex(ignoreRepositories, repository) < 0 {
				versionProperties, err := resolver.StableVersion(versionstream.KindChart, prefix+"/"+release.Name)
				if err != nil {
					return errors.Wrapf(err, "failed to find version number for chart %s", fullChartName)
				}

				version := versionProperties.Version

				versionChanged := false
				if release.Version == "" {
					release.Version = version
					versionChanged = true
				} else if o.UpdateMode && release.Version != version && version != "" {
					release.Version = version
					versionChanged = true
				}
				if versionChanged {
					log.Logger().Infof("resolved chart %s version %s", fullChartName, version)
				}

				if version == "" {
					log.Logger().Warnf("could not find version for chart %s so using latest found in helm repository %s", fullChartName, repository)
				}

				if release.Namespace == "" && versionProperties.Namespace != "" {
					release.Namespace = versionProperties.Namespace
				}
			}
		}

		if release.Namespace == "" && o.Options.Requirements != nil {
			release.Namespace = o.Options.Requirements.Cluster.Namespace
			if release.Namespace == "" {
				release.Namespace = o.Namespace
			}
		}

		releaseNames := []string{chartName}
		if release.Name != "" && release.Name != chartName {
			releaseNames = []string{release.Name, chartName}
		}

		// lets try resolve any values files in the version stream
		found := false
		for _, valueFileName := range valueFileNames {
			versionStreamValuesFile := filepath.Join(versionsDir, "charts", prefix, release.Name, valueFileName)
			exists, err := files.FileExists(versionStreamValuesFile)
			if err != nil {
				return errors.Wrapf(err, "failed to check if version stream values file exists %s", versionStreamValuesFile)
			}
			if exists {
				path := filepath.Join("versionStream", "charts", prefix, release.Name, valueFileName)
				if !valuesContains(release.Values, path) {
					release.Values = append(release.Values, path)
				}
				found = true
				break
			}
			if found {
				break
			}
		}

		// lets try discover any local files
		found = false
		for _, releaseName := range releaseNames {
			for _, valueFileName := range valueFileNames {
				path := filepath.Join("values", releaseName, valueFileName)
				appValuesFile := filepath.Join(appsCfgDir, path)
				exists, err := files.FileExists(appValuesFile)
				if err != nil {
					return errors.Wrapf(err, "failed to check if release values file exists %s", appValuesFile)
				}
				if exists {
					if !valuesContains(release.Values, path) {
						release.Values = append(release.Values, path)
					}
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		helmState.Releases[i] = release
	}

	err = yaml2s.SaveFile(helmState, o.Helmfile)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", o.Helmfile)
	}

	if !o.DoGitCommit {
		return nil
	}
	if count > 0 {
		log.Logger().Infof("committing changes: %s", o.GitCommitMessage)
		err = o.GitCommit(o.Dir, o.GitCommitMessage)
		if err != nil {
			return errors.Wrapf(err, "failed to commit changes")
		}
	}
	return nil
}

// HasHelmfile returns true if there is a helmfile
func (o *Options) HasHelmfile() (bool, error) {
	if o.Helmfile == "" {
		o.Helmfile = filepath.Join(o.Dir, "helmfile.yaml")
	}

	exists, err := files.FileExists(o.Helmfile)
	if err != nil {
		return false, errors.Wrapf(err, "failed to check for file %s", o.Helmfile)
	}
	return exists, nil
}

func valuesContains(values []interface{}, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

// Git returns the gitter - lazily creating one if required
func (o *Options) Git() gitclient.Interface {
	if o.Gitter == nil {
		o.Gitter = cli.NewCLIClient("", o.CommandRunner)
	}
	return o.Gitter
}

func (o *Options) GitCommit(outDir string, commitMessage string) error {
	gitter := o.Git()
	_, err := gitter.Command(outDir, "add", "*")
	if err != nil {
		return errors.Wrapf(err, "failed to add generated resources to git in dir %s", outDir)
	}
	err = gitclient.CommitIfChanges(gitter, outDir, commitMessage)
	if err != nil {
		return errors.Wrapf(err, "failed to commit changes to git in dir %s", outDir)
	}
	return nil
}

// CustomUpgrades performs custom upgrades outside of the version stream/kpt approach
func (o *Options) CustomUpgrades() error {
	requirements, fileName, err := config.LoadRequirementsConfig(o.Dir, false)
	if err != nil {
		return errors.Wrapf(err, "failed to load the requirements configuration")
	}

	if requirements.BuildPacks == nil {
		requirements.BuildPacks = &config.BuildPackConfig{}
	}
	if requirements.BuildPacks.BuildPackLibrary == nil {
		requirements.BuildPacks.BuildPackLibrary = &config.BuildPackLibrary{}
	}

	gitURL := requirements.BuildPacks.BuildPackLibrary.GitURL
	if gitURL == "" || strings.HasPrefix(gitURL, "https://github.com/jenkins-x/jxr-packs-kubernetes") {
		requirements.BuildPacks.BuildPackLibrary.GitURL = "https://github.com/jenkins-x/jx3-pipeline-catalog.git"

		err = requirements.SaveConfig(fileName)
		if err != nil {
			return errors.Wrapf(err, "failed to save requirements file %s", fileName)
		}

		log.Logger().Infof("updated the build pack library to be %s", termcolor.ColorInfo(requirements.BuildPacks.BuildPackLibrary.GitURL))
	}

	// lets replace the old tekton repositories if they are being used
	for i := range o.Results.HelmState.Repositories {
		repo := &o.Results.HelmState.Repositories[i]
		if strings.TrimSuffix(repo.URL, "/") == "https://kubernetes-charts.storage.googleapis.com" {
			repo.URL = "https://charts.helm.sh/stable"
			break
		}
	}

	// lets replace the old tekton chart if its being used
	for i := range o.Results.HelmState.Releases {
		release := &o.Results.HelmState.Releases[i]
		if release.Chart == "jenkins-x/tekton" {
			release.Chart = "cdf/tekton-pipeline"
			release.Namespace = "tekton-pipelines"

			// lets make sure we have a cdf repository
			found := false
			for _, repo := range o.Results.HelmState.Repositories {
				if repo.Name == "cdf" {
					found = true
					break
				}
			}
			if !found {
				o.Results.HelmState.Repositories = append(o.Results.HelmState.Repositories, state.RepositorySpec{
					Name: "cdf",
					URL:  "https://cdfoundation.github.io/tekton-helm-chart",
				})
			}
			break
		}
	}

	// lets replace the old chartmuseum chart if its being used
	for i := range o.Results.HelmState.Releases {
		release := &o.Results.HelmState.Releases[i]
		if release.Chart == "jenkins-x/chartmuseum" {
			release.Chart = "stable/chartmuseum"
			o.updateVersionFromVersionStream(release)
			release.Values = []interface{}{"versionStream/charts/stable/chartmuseum/values.yaml.gotmpl"}

			// lets make sure we have a cdf repository
			found := false
			for _, repo := range o.Results.HelmState.Repositories {
				if repo.Name == "stable" {
					found = true
					break
				}
			}
			if !found {
				o.Results.HelmState.Repositories = append(o.Results.HelmState.Repositories, state.RepositorySpec{
					Name: "stable",
					URL:  "https://charts.helm.sh/stable",
				})
			}
			break
		}
	}
	ns := requirements.Cluster.Namespace
	if ns == "" {
		ns = "jx"
	}

	if requirements.SecretStorage == config.SecretStorageTypeLocal {
		// lets make sure the local external secrets chart is included
		found := false
		for i := range o.Results.HelmState.Releases {
			release := &o.Results.HelmState.Releases[i]
			if release.Chart == "jx3/local-external-secrets" {
				found = true
				break
			}
		}
		if !found {
			release := state.ReleaseSpec{
				Chart:     "jx3/local-external-secrets",
				Namespace: ns,
			}
			o.updateVersionFromVersionStream(&release)
			o.Results.HelmState.Releases = append(o.Results.HelmState.Releases, release)
		}
	}

	// lets replace the old jx-labs/ charts...
	for _, name := range []string{"jenkins-x-crds", "pusher-wave", "vault-instance"} {
		chartName := "jx-labs/" + name
		for i := range o.Results.HelmState.Releases {
			release := &o.Results.HelmState.Releases[i]
			if release.Chart == chartName {
				release.Chart = "jx3/" + name
				if name == "jenkins-x-crds" {
					release.Values = []interface{}{"versionStream/charts/jx3/jenkins-x-crds/values.yaml.gotmpl"}
				}
				o.updateVersionFromVersionStream(release)
				break
			}
		}
	}

	// remove jx-labs repository if we have no more charts left using the prefix
	jxLabsCount := 0
	for i := range o.Results.HelmState.Releases {
		release := &o.Results.HelmState.Releases[i]
		if strings.HasPrefix(release.Chart, "jx-labs/") {
			jxLabsCount++
		}
	}
	if jxLabsCount == 0 {
		for i := range o.Results.HelmState.Repositories {
			if o.Results.HelmState.Repositories[i].Name == "jx-labs" {
				o.Results.HelmState.Repositories = append(o.Results.HelmState.Repositories[0:i], o.Results.HelmState.Repositories[i+1:]...)
				break
			}
		}
	}

	// lets ensure we have the jx-build-controller installed
	found := false
	for i := range o.Results.HelmState.Releases {
		release := &o.Results.HelmState.Releases[i]
		if release.Chart == "jx3/jx-build-controller" {
			found = true
			break
		}
	}
	if !found {
		o.Results.HelmState.Releases = append(o.Results.HelmState.Releases, state.ReleaseSpec{
			Chart:     "jx3/jx-build-controller",
			Namespace: ns,
		})

		// lets make sure we have a jx3 repository
		found = false
		for _, repo := range o.Results.HelmState.Repositories {
			if repo.Name == "jx3" {
				found = true
				break
			}
		}
		if !found {
			o.Results.HelmState.Repositories = append(o.Results.HelmState.Repositories, state.RepositorySpec{
				Name: "jx3",
				URL:  "https://storage.googleapis.com/jenkinsxio/charts",
			})
		}
	}

	// TODO lets remove the jx-labs repository if its no longer referenced...

	lighthouseTriggerFile := filepath.Join(o.Dir, ".lighthouse", "jenkins-x", "triggers.yaml")
	exists, err := files.FileExists(lighthouseTriggerFile)
	if err != nil {
		return errors.Wrapf(err, "failed to detect file %s", lighthouseTriggerFile)
	}
	if !exists {
		bin := o.KptBinary
		if bin == "" {
			bin, err = plugins.GetKptBinary(plugins.KptVersion)
			if err != nil {
				return err
			}
		}

		args := []string{"pkg", "get", "https://github.com/jenkins-x/jx3-pipeline-catalog.git/environment/.lighthouse", o.Dir}
		c := &cmdrunner.Command{
			Name: bin,
			Args: args,
			Dir:  o.Dir,
		}
		_, err = o.CommandRunner(c)
		if err != nil {
			return errors.Wrapf(err, "failed to get environment tekton pipeline via kpt in dir %s", o.Dir)
		}

		err = gitclient.Add(o.Git(), o.Dir, ".lighthouse")
		if err != nil {
			return errors.Wrapf(err, "failed to add .lighthouse dir to git")
		}

		log.Logger().Infof("got tekton pipeline for envirnment at %s", lighthouseTriggerFile)
	}
	return nil
}

func (o *Options) updateVersionFromVersionStream(release *state.ReleaseSpec) {
	versionProperties, err := o.Options.Resolver.StableVersion(versionstream.KindChart, release.Chart)
	if err != nil {
		log.Logger().Warnf("failed to find version number for chart %s", release.Chart)
		release.Version = ""
	}
	release.Version = versionProperties.Version
}
