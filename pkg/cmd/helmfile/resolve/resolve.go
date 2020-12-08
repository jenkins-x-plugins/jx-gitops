package resolve

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/jxtmpl/reqvalues"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/structure"
	"github.com/jenkins-x/jx-gitops/pkg/helmhelpers"
	"github.com/jenkins-x/jx-gitops/pkg/plugins"
	"github.com/jenkins-x/jx-gitops/pkg/versionstreamer"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
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

const (
	versionStreamDir = "versionStream"
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
	pathSeparator  = string(os.PathSeparator)
)

type Helmfile struct {
	filepath           string
	relativePathToRoot string
}

// Options the options for the command
type Options struct {
	versionstreamer.Options
	Namespace        string
	GitCommitMessage string
	Helmfile         string
	Helmfiles        []Helmfile
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
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", "jx", "the default namespace if none is specified in the helmfile.yaml")

	// git commit stuff....
	cmd.Flags().BoolVarP(&o.DoGitCommit, prefix+"git-commit", "", false, "if set then the template command will git commit the modified helmfile.yaml files")
}

func gatherHelmfiles(helmfile string) ([]Helmfile, error) {
	helmState := state.HelmState{}
	err := yaml2s.LoadFile(helmfile, &helmState)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load helmfile %s", helmfile)
	}

	helmfiles := []Helmfile{
		{helmfile, ""},
	}
	parentHelmfileDir := filepath.Dir(helmfile)
	for _, nested := range helmState.Helmfiles {
		nestedHelmfileDepth := len(strings.Split(filepath.Dir(nested.Path), pathSeparator))
		relativePath := strings.Repeat("../", nestedHelmfileDepth)

		helmfiles = append(helmfiles, Helmfile{filepath.Join(parentHelmfileDir, nested.Path), relativePath})
	}
	return helmfiles, nil
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

	helmfiles, err := gatherHelmfiles(o.Helmfile)
	if err != nil {
		return errors.Wrapf(err, "failed to gather nested helmfiles")
	}
	o.Helmfiles = helmfiles

	if o.GitCommitMessage == "" {
		o.GitCommitMessage = "chore: resolved charts and values from the version stream"
	}

	o.prefixes, err = o.Options.Resolver.GetRepositoryPrefixes()
	if err != nil {
		return errors.Wrapf(err, "failed to load repository prefixes at %s", o.VersionStreamDir)
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

	resolver := o.Options.Resolver
	if resolver == nil {
		return errors.Errorf("failed to create the VersionResolver")
	}

	count := 0

	if o.UpdateMode {
		increment, err := o.upgradeHelmfileStructure(o.Dir)
		count += increment
		if err != nil {
			return errors.Wrapf(err, "failed to perform custom upgrades")
		}
	}

	helmfiles, err := gatherHelmfiles(o.Helmfile)
	if err != nil {
		return errors.Wrapf(err, "error gathering helmfiles")
	}

	for _, helmfile := range helmfiles {
		increment, err := o.processHelmfile(helmfile)
		if err != nil {
			return errors.Wrapf(err, "failed to process helmfile %s", helmfile.filepath)
		}
		count += increment
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

func (o *Options) processHelmfile(helmfile Helmfile) (int, error) {
	helmState := state.HelmState{}
	path := helmfile.filepath
	err := yaml2s.LoadFile(path, &helmState)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to load helmfile %s", helmfile)
	}

	if o.UpdateMode {
		err = o.CustomUpgrades(&helmState)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to perform custom upgrades")
		}
	}

	if helmfile.relativePathToRoot != "" {
		helmfileDir := filepath.Dir(path)
		jxReqValuesFileName := filepath.Join(helmfileDir, reqvalues.RequirementsValuesFileName)
		o.Results.RequirementsValuesFileName = reqvalues.RequirementsValuesFileName
		requirements := *o.Options.Requirements
		if err != nil {
			return 0, errors.Wrapf(err, "failed to save tempo file for jx requirements values file %s", jxReqValuesFileName)
		}
		ns := helmState.OverrideNamespace
		if ns == "" {
			_, ns = filepath.Split(helmfileDir)
		}
		requirements.Ingress.NamespaceSubDomain = strings.ReplaceAll(requirements.Ingress.NamespaceSubDomain, "jx", ns)
		err = reqvalues.SaveRequirementsValuesFile(&requirements, o.Dir, jxReqValuesFileName)
	}

	increment, err := o.resolveHelmfile(&helmState, helmfile)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to resolve helmfile %s", helmfile)
	}

	err = yaml2s.SaveFile(helmState, path)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to save file %s", helmfile)
	}
	return increment, nil
}

func (o *Options) upgradeHelmfileStructure(dir string) (int, error) {
	if exists, _ := files.DirExists(filepath.Join(dir, structure.HelmfileFolder)); exists {
		return 0, nil
	}
	// First let's resolve parent helmfile before restructuring
	count := 0
	increment, err := o.processHelmfile(o.Helmfiles[0])
	if err != nil {
		return 0, errors.Wrapf(err, "error processing parent helmfile before restructure")
	}

	count += increment
	so := structure.Options{
		Dir: dir,
	}
	err = so.Run()
	if err != nil {
		return 0, errors.Wrapf(err, "error restructuring helmfiles during resolve upgrade")
	}
	count++

	helmfiles, err := gatherHelmfiles(o.Helmfile)
	if err != nil {
		return 0, errors.Wrapf(err, "error gathering helmfiles")
	}
	o.Helmfiles = helmfiles
	return count, nil
}

func (o *Options) resolveHelmfile(helmState *state.HelmState, helmfile Helmfile) (int, error) {
	var err error
	var ignoreRepositories []string
	if !helmhelpers.IsInCluster() || o.TestOutOfCluster {
		ignoreRepositories, err = helmhelpers.FindClusterLocalRepositoryURLs(helmState.Repositories)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to find cluster local repositories")
		}
	}

	err = helmhelpers.AddHelmRepositories(o.HelmBinary, *helmState, o.QuietCommandRunner, ignoreRepositories)
	if err != nil {
		return 0, errors.Wrapf(err, "failed to add helm repositories")
	}

	if helmfile.relativePathToRoot != "" {
		// ensure we have added the jx-values.yaml file in the envirionment
		if helmState.Environments == nil {
			helmState.Environments = map[string]state.EnvironmentSpec{}
		}
		// lets remove any old legacy files in the root dir
		oldFiles := []string{
			filepath.Join("..", "..", reqvalues.RequirementsValuesFileName),
			filepath.Join("..", "..", "versionStream", "src", "fake-secrets.yaml.gotmpl"),
		}
		envSpec := helmState.Environments["default"]
		for _, f := range oldFiles {
			for i, v := range envSpec.Values {
				s, ok := v.(string)
				if ok && s == f {
					newValues := envSpec.Values[0:i]
					if len(envSpec.Values) > i+1 {
						newValues = append(newValues, envSpec.Values[i+1:]...)
					}
					envSpec.Values = newValues
					helmState.Environments["default"] = envSpec
					break
				}
			}
		}

		envSpec = helmState.Environments["default"]
		foundValuesFile := false
		for _, v := range envSpec.Values {
			s, ok := v.(string)
			if ok && s == reqvalues.RequirementsValuesFileName {
				foundValuesFile = true
				break
			}
		}
		if !foundValuesFile {
			envValue := helmState.Environments["default"]
			envValue.Values = append(envValue.Values, reqvalues.RequirementsValuesFileName)
			helmState.Environments["default"] = envValue
		}
	}

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
			if prefix != "" {
				for _, r := range helmState.Repositories {
					if r.Name == prefix {
						repository = r.URL
					}
				}
			}
			if repository == "" && prefix != "" {
				repository, err = versionstreamer.MatchRepositoryPrefix(o.prefixes, prefix)
				if err != nil {
					return 0, errors.Wrapf(err, "failed to match prefix %s with repositories from versionstream %s", prefix, o.VersionStreamURL)
				}
			}
			if repository == "" && prefix != "" {
				return 0, errors.Wrapf(err, "failed to find repository URL, not defined in helmfile.yaml or versionstream %s", o.VersionStreamURL)
			}
			if repository != "" && prefix != "" {
				// lets ensure we've got a repository for this URL in the apps file
				found := false
				for _, r := range helmState.Repositories {
					if r.Name == prefix {
						if r.URL != repository {
							return 0, errors.Errorf("release %s has prefix %s for repository URL %s which is also mapped to prefix %s", release.Name, prefix, r.URL, r.Name)
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
				// first try and match using the prefix and release name as we might have a version stream folder that uses helm alias
				versionProperties, err := o.Options.Resolver.StableVersion(versionstream.KindChart, prefix+"/"+release.Name)
				if err != nil {
					return 0, errors.Wrapf(err, "failed to find version number for chart %s", release.Name)
				}

				// lets fall back to using the full chart name
				if versionProperties.Version == "" {
					versionProperties, err = o.Options.Resolver.StableVersion(versionstream.KindChart, fullChartName)
					if err != nil {
						return 0, errors.Wrapf(err, "failed to find version number for chart %s", fullChartName)
					}
				}

				version := versionProperties.Version

				if release.Version == "" && version == "" {
					log.Logger().Warnf("could not find version for chart %s so using latest found in helm repository %s", fullChartName, repository)
				}

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

				if release.Namespace == "" && helmState.OverrideNamespace == "" && versionProperties.Namespace != "" {
					release.Namespace = versionProperties.Namespace
				}
			}
		}

		if release.Namespace == "" && helmState.OverrideNamespace == "" {
			release.Namespace = o.Namespace
			if release.Namespace == "" {
				release.Namespace = jxcore.DefaultNamespace
			}
		}

		releaseNames := []string{chartName}
		if release.Name != "" && release.Name != chartName {
			releaseNames = []string{release.Name, chartName}
		}

		// lets try resolve any values files in the version stream using the prefix and chart name first
		found, err := o.addValues(helmfile, filepath.Join(prefix, release.Name), &release)
		if err != nil {
			return 0, errors.Wrapf(err, "failed to add values")
		}
		if !found {
			// next try the full chart name
			found, err = o.addValues(helmfile, fullChartName, &release)
			if err != nil {
				return 0, errors.Wrapf(err, "failed to add values")
			}
		}

		// lets try discover any local files
		found = false
		for _, releaseName := range releaseNames {
			for _, valueFileName := range valueFileNames {
				path := filepath.Join(helmfile.relativePathToRoot, "values", releaseName, valueFileName)
				appValuesFile := filepath.Join(o.Dir, path)
				exists, err := files.FileExists(appValuesFile)
				if err != nil {
					return 0, errors.Wrapf(err, "failed to check if release values file exists %s", appValuesFile)
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

		if helmfile.relativePathToRoot != "" {
			foundValuesFile := false
			for _, v := range release.Values {
				s, ok := v.(string)
				if ok && s == reqvalues.RequirementsValuesFileName {
					foundValuesFile = true
					break
				}
			}
			if !foundValuesFile {
				release.Values = append(release.Values, reqvalues.RequirementsValuesFileName)
			}
		}
		helmState.Releases[i] = release
	}

	return count, nil

}

func (o *Options) addValues(helmfile Helmfile, name string, release *state.ReleaseSpec) (bool, error) {
	found := false
	for _, valueFileName := range valueFileNames {
		versionStreamValuesFile := filepath.Join(o.Resolver.VersionsDir, "charts", name, valueFileName)
		exists, err := files.FileExists(versionStreamValuesFile)
		if err != nil {
			return false, errors.Wrapf(err, "failed to check if version stream values file exists %s", versionStreamValuesFile)
		}
		if exists {
			path := filepath.Join(helmfile.relativePathToRoot, versionStreamDir, "charts", name, valueFileName)
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
	return found, nil
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
func (o *Options) CustomUpgrades(helmstate *state.HelmState) error {
	err := o.migrateRequirementsToV4()
	if err != nil {
		return errors.Wrapf(err, "failed to migrate jx-requirements.yml")
	}
	err = o.renameImagePullSecretsFile()
	if err != nil {
		return errors.Wrapf(err, "failed to rename old image pull secrets file")
	}

	var versionStreamPath string
	if helmstate.OverrideNamespace == "" {
		versionStreamPath = "versionStream"
	} else {
		versionStreamPath = "../../versionStream"

		// lets remove the old top level jx-values.yaml as we are using multi-level helmfiles
		oldJXValues := filepath.Join(o.Dir, "jx-values.yaml")
		exists, err := files.FileExists(oldJXValues)
		if err != nil {
			return errors.Wrapf(err, "failed to check if file exists %s", oldJXValues)
		}
		if exists {
			err = os.Remove(oldJXValues)
			if err != nil {
				return errors.Wrapf(err, "failed to remove old file %s", oldJXValues)
			}
		}
	}

	requirementsResource, _, err := jxcore.LoadRequirementsConfig(o.Dir, false)
	if err != nil {
		return errors.Wrapf(err, "failed to load the requirements configuration")
	}
	requirements := &requirementsResource.Spec
	// lets replace the old tekton repositories if they are being used
	for i := range helmstate.Repositories {
		repo := &helmstate.Repositories[i]
		cleanURL := strings.TrimSuffix(repo.URL, "/")
		switch cleanURL {
		case "https://kubernetes-charts.storage.googleapis.com":
			repo.URL = "https://charts.helm.sh/stable"
		case "https://godaddy.github.io/kubernetes-external-secrets":
			repo.URL = "https://external-secrets.github.io/kubernetes-external-secrets"
		}
	}

	// lets replace the old tekton chart if its being used
	for i := range helmstate.Releases {
		release := &helmstate.Releases[i]
		if release.Chart == "jenkins-x/tekton" {
			release.Chart = "cdf/tekton-pipeline"
			release.Namespace = "tekton-pipelines"

			// lets make sure we have a cdf repository
			found := false
			for _, repo := range helmstate.Repositories {
				if repo.Name == "cdf" {
					found = true
					break
				}
			}
			if !found {
				helmstate.Repositories = append(helmstate.Repositories, state.RepositorySpec{
					Name: "cdf",
					URL:  "https://cdfoundation.github.io/tekton-helm-chart",
				})
			}
			break
		}
	}

	// lets replace the old chartmuseum chart if its being used
	for i := range helmstate.Releases {
		release := &helmstate.Releases[i]
		if release.Chart == "jenkins-x/chartmuseum" {
			release.Chart = "stable/chartmuseum"
			o.updateVersionFromVersionStream(release)
			release.Values = []interface{}{fmt.Sprintf("%s/charts/stable/chartmuseum/values.yaml.gotmpl", versionStreamPath)}

			// lets make sure we have a cdf repository
			found := false
			for _, repo := range helmstate.Repositories {
				if repo.Name == "stable" {
					found = true
					break
				}
			}
			if !found {
				helmstate.Repositories = append(helmstate.Repositories, state.RepositorySpec{
					Name: "stable",
					URL:  "https://charts.helm.sh/stable",
				})
			}
			break
		}
	}

	// Replace old nginx ingress chart
	for i := range helmstate.Releases {
		release := &helmstate.Releases[i]
		switch release.Chart {
		case "stable/nginx-ingress":
			release.Chart = "ingress-nginx/ingress-nginx"
			o.updateVersionFromVersionStream(release)
			release.Values = []interface{}{fmt.Sprintf("%s/charts/ingress-nginx/ingress-nginx/values.yaml.gotmpl", versionStreamPath)}

			// lets make sure we have the ingress-nginx repository
			found := false
			for _, repo := range helmstate.Repositories {
				if repo.Name == "ingress-nginx" {
					found = true
					break
				}
			}
			if !found {
				helmstate.Repositories = append(helmstate.Repositories, state.RepositorySpec{
					Name: "ingress-nginx",
					URL:  "https://kubernetes.github.io/ingress-nginx",
				})
			}
			break
		case "ingress-nginx/ingress-nginx":
			for i := range release.Values {
				v := release.Values[i]
				s, ok := v.(string)
				// lets switch invalid paths to the one inside a chart repo folder
				if ok && s == fmt.Sprintf("%s/charts/ingress-nginx/values.yaml.gotmpl", versionStreamPath) {
					release.Values[i] = fmt.Sprintf("%s/charts/ingress-nginx/ingress-nginx/values.yaml.gotmpl", versionStreamPath)
					break
				}
			}
		}
	}

	ns := jxcore.DefaultNamespace

	if requirements.SecretStorage == jxcore.SecretStorageTypeLocal && helmstate.OverrideNamespace == ns {
		// lets make sure the local external secrets chart is included
		found := false
		for i := range helmstate.Releases {
			release := &helmstate.Releases[i]
			if release.Chart == "jx3/local-external-secrets" {
				found = true
				break
			}
		}
		if !found {
			release := state.ReleaseSpec{
				Chart: "jx3/local-external-secrets",
			}
			o.updateVersionFromVersionStream(&release)
			helmstate.Releases = append(helmstate.Releases, release)
		}
	}

	// lets replace the old jx-labs/ charts...
	for _, name := range []string{"jenkins-x-crds", "pusher-wave", "vault-instance"} {
		chartName := "jx-labs/" + name
		for i := range helmstate.Releases {
			release := &helmstate.Releases[i]
			if release.Chart == chartName {
				release.Chart = "jx3/" + name
				if name == "jenkins-x-crds" {
					release.Values = []interface{}{fmt.Sprintf("%s/charts/jx3/jenkins-x-crds/values.yaml.gotmpl", versionStreamPath)}
				}
				o.updateVersionFromVersionStream(release)
				break
			}
		}
	}

	// remove jx-labs repository if we have no more charts left using the prefix
	jxLabsCount := 0
	for i := range helmstate.Releases {
		release := &helmstate.Releases[i]
		if strings.HasPrefix(release.Chart, "jx-labs/") {
			jxLabsCount++
		}
	}
	if jxLabsCount == 0 {
		for i := range helmstate.Repositories {
			if helmstate.Repositories[i].Name == "jx-labs" {
				helmstate.Repositories = append(helmstate.Repositories[0:i], helmstate.Repositories[i+1:]...)
				break
			}
		}
	}

	// lets ensure we have the jx-build-controller installed
	found := false
	for i := range helmstate.Releases {
		release := &helmstate.Releases[i]
		if release.Chart == "jx3/jx-build-controller" {
			found = true
			break
		}
	}
	if !found && helmstate.OverrideNamespace == "jx" {
		helmstate.Releases = append(helmstate.Releases, state.ReleaseSpec{
			Chart: "jx3/jx-build-controller",
		})

		// lets make sure we have a jx3 repository
		found = false
		for _, repo := range helmstate.Repositories {
			if repo.Name == "jx3" {
				found = true
				break
			}
		}
		if !found {
			helmstate.Repositories = append(helmstate.Repositories, state.RepositorySpec{
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

	return o.customBootJob(requirements)
}

func (o *Options) migrateRequirementsToV4() error {
	path := filepath.Join(o.Dir, "jx-requirements.yml")
	exists, err := files.FileExists(path)
	if err != nil {
		return errors.Wrapf(err, "failed checking if jx-requirements.yml exists")
	}
	if !exists {
		return fmt.Errorf("failed to migrate jx-requirements.yml as it does not exist in directory %s", o.Dir)
	}

	if exists {
		file, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to read %s", path)
		}

		if !jxcore.IsNewRequirementsFile(string(file)) {
			log.Logger().Info(termcolor.ColorInfo("Migrating your jx-requirements.yml file, please ignore warnings about validation failures in YAML"))

			reqs, err := jxcore.LoadRequirementsConfigFileNoDefaults(path, false)
			if err != nil {
				return errors.Wrapf(err, "failed loading jx-requirements.yml in directory %s", o.Dir)
			}
			err = reqs.SaveConfig(path)
			if err != nil {
				return errors.Wrap(err, "failed checking if jx-requirements.yml exists")
			}
		}

	}

	return nil
}

func (o *Options) customBootJob(requirements *jxcore.RequirementsConfig) error {
	secretKind := requirements.SecretStorage
	secretText := string(secretKind)
	if secretText == "" {
		secretKind = jxcore.SecretStorageTypeLocal
	}
	jobFileName := "job.yaml"
	if secretText != "local" {
		name := "job-" + secretText + ".yaml"
		fileName := filepath.Join(o.Dir, "versionStream", "git-operator", name)
		exists, err := files.FileExists(fileName)
		if err != nil {
			return errors.Wrapf(err, "failed to check if file exists %s", fileName)
		}
		if !exists {
			log.Logger().Warnf("file does not exist %s so defaulting to boot job file: job.yaml", fileName)
		} else {
			jobFileName = name
		}
	}
	dir := filepath.Join(o.Dir, ".jx", "git-operator")
	err := os.MkdirAll(dir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create dir %s", dir)
	}
	f := filepath.Join(dir, "filename.txt")
	err = ioutil.WriteFile(f, []byte(jobFileName), files.DefaultFileWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to write file %s", f)
	}
	return nil
}

func (o *Options) updateVersionFromVersionStream(release *state.ReleaseSpec) {
	versionProperties, err := o.Options.Resolver.StableVersion(versionstream.KindChart, release.Chart)
	if err != nil {
		log.Logger().Warnf("failed to find version number for chart %s", release.Chart)
		release.Version = ""
	}

	if versionProperties == nil {
		log.Logger().Warnf("failed to find version number for chart %s", release.Chart)
		release.Version = ""
		return
	}

	release.Version = versionProperties.Version
}

func (o *Options) renameImagePullSecretsFile() error {
	oldPath := filepath.Join(o.Dir, "imagePullSecrets.yaml")
	newPath := filepath.Join(o.Dir, "jx-global-values.yaml")
	exists, err := files.FileExists(oldPath)
	if err != nil {
		return errors.Wrapf(err, "failed to check for %s", oldPath)
	}
	if !exists {
		return nil
	}
	err = os.Rename(oldPath, newPath)
	if err != nil {
		return errors.Wrapf(err, "failed to rename %s to %s", oldPath, newPath)
	}
	return nil
}
