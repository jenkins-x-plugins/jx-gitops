package resolve

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/helmfiles"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/jxtmpl/reqvalues"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/pipelinecatalogs"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/quickstarthelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"

	"github.com/helmfile/helmfile/pkg/state"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helmfile/structure"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/helmhelpers"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/plugins"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/versionstreamer"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/versionstream"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	versionStreamDir = "versionStream"

	useHelmfileRepos = false
)

var (
	info = termcolor.ColorInfo

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
	Namespace               string
	GitCommitMessage        string
	Helmfile                string
	Helmfiles               []helmfiles.Helmfile
	KptBinary               string
	HelmfileBinary          string
	HelmBinary              string
	BatchMode               bool
	UpdateMode              bool
	DoGitCommit             bool
	TestOutOfCluster        bool
	Gitter                  gitclient.Interface
	prefixes                *versionstream.RepositoryPrefixes
	Results                 Results
	AddEnvironmentPipelines bool
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
		Run: func(_ *cobra.Command, _ []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.BaseOptions.AddBaseFlags(cmd)

	cmd.Flags().BoolVarP(&o.UpdateMode, "update", "", false, "updates versions from the version stream if they have changed")
	if useHelmfileRepos {
		cmd.Flags().StringVarP(&o.HelmfileBinary, "helmfile-binary", "", "", "specifies the helmfile binary location to use. If not specified defaults to using the downloaded helmfile plugin")
	}
	cmd.Flags().StringVarP(&o.HelmBinary, "helm-binary", "", "", "specifies the helm binary location to use. If not specified defaults to using the downloaded helm plugin")
	o.AddFlags(cmd, "")
	return cmd, o
}

func (o *Options) AddFlags(cmd *cobra.Command, prefix string) {
	cmd.Flags().StringVarP(&o.Helmfile, "helmfile", "", "", "the helmfile to resolve. If not specified defaults to 'helmfile.yaml' in the dir")
	cmd.Flags().StringVarP(&o.GitCommitMessage, prefix+"commit-message", "", "chore: generated kubernetes resources from helm chart", "the git commit message used")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", "jx", "the default namespace if none is specified in the helmfile.yaml")
	cmd.Flags().BoolVarP(&o.AddEnvironmentPipelines, "add-environment-pipelines", "", false, "skips the custom upgrade step for adding .lighthouse folder")

	// git commit stuff....
	cmd.Flags().BoolVarP(&o.DoGitCommit, prefix+"git-commit", "", false, "if set then the template command will git commit the modified helmfile.yaml files")

	cmd.Flags().StringVarP(&o.VersionStreamDir, "version-stream-dir", "", "", "the directory for the version stream. Defaults to 'versionStream' in the current --dir")
}

// Validate validates the options and populates any missing values
func (o *Options) Validate() error {
	err := o.Options.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to ")
	}

	if o.Helmfile == "" {
		o.Helmfile = "helmfile.yaml"
	}

	if useHelmfileRepos {
		if o.HelmfileBinary == "" {
			o.HelmfileBinary, err = plugins.GetHelmfileBinary(plugins.HelmfileVersion)
			if err != nil {
				return errors.Wrapf(err, "failed to download helmfile plugin")
			}
		}
	}
	if o.HelmBinary == "" {
		o.HelmBinary, err = plugins.GetHelmBinary(plugins.HelmVersion)
		if err != nil {
			return errors.Wrapf(err, "failed to download helm plugin")
		}
	}

	if o.Dir == "" {
		o.Dir = "."
	}
	helmfiles, err := helmfiles.GatherHelmfiles(o.Helmfile, o.Dir)
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

		err = o.upgradePipelineCatalog()
		if err != nil {
			return errors.Wrapf(err, "failed to upgrade pipeline catalog")
		}
	}

	if useHelmfileRepos {
		// lets add the helm repositories
		args := []string{}
		if o.HelmBinary != "" {
			args = append(args, "--helm-binary", o.HelmBinary)
		}
		args = append(args, "repos")
		c := &cmdrunner.Command{
			Dir:  o.Dir,
			Name: o.HelmfileBinary,
			Args: args,
		}
		_, err = o.QuietCommandRunner(c)
		if err != nil {
			return errors.Wrapf(err, "failed to run command %s in dir %s", c.CLI(), o.Dir)
		}
	}

	includedHelmfiles, err := helmfiles.GatherHelmfiles(o.Helmfile, o.Dir)
	if err != nil {
		return errors.Wrapf(err, "error gathering helmfiles")
	}

	for _, helmfile := range includedHelmfiles {
		err := o.processHelmfile(helmfile)
		if err != nil {
			return errors.Wrapf(err, "failed to process helmfile %s", helmfile.Filepath)
		}
		// ToDo: What are we trying to do here?
		count += 0
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

// Handle WARNING: environments and releases cannot be defined within the same YAML part. Use --- to extract the environments into a dedicated part
// Split automatically
// Handle helmfile with multiple documents.
func (o *Options) processHelmfile(helmfile helmfiles.Helmfile) error {
	path := helmfile.Filepath
	helmStates, err := helmfiles.LoadHelmfile(path)
	if err != nil {
		return err
	}

	for _, helmState := range helmStates {
		if o.UpdateMode {
			err = o.CustomUpgrades(helmState)
			if err != nil {
				return errors.Wrapf(err, "failed to perform custom upgrades")
			}
		}

		err = o.resolveHelmfile(helmState, helmfile)
		if err != nil {
			return errors.Wrapf(err, "failed to resolve helmfile %s", helmfile)
		}

		if o.UpdateMode {
			// let's remove any unused chart repositories
			removeRedundantRepositories(helmState)
		}
	}

	if helmfile.RelativePathToRoot != "" {
		// Split file
		if len(helmStates) == 1 {
			environmentHead := []*state.HelmState{
				{
					ReleaseSetSpec: state.ReleaseSetSpec{
						Environments: helmStates[0].Environments,
					},
				},
			}
			helmStates[0].Environments = nil
			helmStates = append(environmentHead, helmStates...)
		}
		o.ensureEnvironment(helmStates[0], helmfile)
		helmfileDir := filepath.Dir(path)
		ns := helmStates[1].OverrideNamespace
		if ns == "" {
			_, ns = filepath.Split(helmfileDir)
		}
		err := o.saveNamespaceJXValuesFile(helmfileDir, ns)
		if err != nil {
			return errors.Wrapf(err, "failed to generate jx-values.yaml for namespace %s", ns)
		}
	}

	return helmfiles.SaveHelmfile(path, helmStates)
}

func (o *Options) saveNamespaceJXValuesFile(helmfileDir, ns string) error {
	jxReqValuesFileName := filepath.Join(helmfileDir, reqvalues.RequirementsValuesFileName)
	o.Results.RequirementsValuesFileName = reqvalues.RequirementsValuesFileName
	requirements := *o.Options.Requirements
	subDomain := strings.ReplaceAll(requirements.Ingress.NamespaceSubDomain, "jx", ns)
	requirements.Ingress.NamespaceSubDomain = subDomain

	envKey := ""
	for k := range requirements.Environments {
		e := requirements.Environments[k]
		if (ns == e.Namespace) || (strings.TrimPrefix(ns, "jx-") == e.Key) {
			envKey = e.Key
			break
		}
	}

	// lets see if there is a custom ingress value for this namespace
	for k := range requirements.Environments {
		e := requirements.Environments[k]
		if e.Ingress != nil && e.Key == envKey {
			requirements.Ingress = *e.Ingress
			if requirements.Ingress.NamespaceSubDomain == "" {
				requirements.Ingress.NamespaceSubDomain = subDomain
			}
			if requirements.Ingress.Domain == "" {
				requirements.Ingress.Domain = o.Options.Requirements.Ingress.Domain
			}
		}
	}

	// lets make sure we use a default domain name to avoid the validation of the Ingress
	// resources from failing
	if requirements.Ingress.Domain == "" {
		requirements.Ingress.Domain = v1alpha1.DomainPlaceholder
	}

	err := reqvalues.SaveRequirementsValuesFile(&requirements, o.Dir, jxReqValuesFileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save jx-values.yaml file")
	}
	return nil
}

func (o *Options) upgradeHelmfileStructure(dir string) (int, error) {
	count := 0
	err := o.processHelmfile(o.Helmfiles[0])
	if err != nil {
		return 0, errors.Wrapf(err, "error processing parent helmfile before restructure")
	}

	count += 0

	if exists, _ := files.DirExists(filepath.Join(dir, structure.HelmfileFolder)); !exists {
		so := structure.Options{
			Dir: dir,
		}
		err = so.Run()
		if err != nil {
			return 0, errors.Wrapf(err, "error restructuring helmfiles during resolve upgrade")
		}
	}
	count++

	includedHelmfiles, err := helmfiles.GatherHelmfiles(o.Helmfile, o.Dir)
	if err != nil {
		return 0, errors.Wrapf(err, "error gathering helmfiles")
	}
	o.Helmfiles = includedHelmfiles

	err = gitclient.Add(o.Git(), o.Dir, "helmfiles")
	if err != nil {
		return count, errors.Wrapf(err, "failed to add files to git")
	}
	return count, nil
}

func (o *Options) resolveHelmfile(helmState *state.HelmState, helmfile helmfiles.Helmfile) error {
	if helmState.Releases == nil {
		return nil
	}
	var err error
	var ignoreRepositories []string
	if !helmhelpers.IsInCluster() || o.TestOutOfCluster {
		ignoreRepositories, err = helmhelpers.FindClusterLocalRepositoryURLs(helmState.Repositories)
		if err != nil {
			return errors.Wrapf(err, "failed to find cluster local repositories")
		}
	}
	for i := range helmState.Releases {
		release := helmState.Releases[i]
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
		// ignore remote charts
		if strings.Contains(fullChartName, "::") {
			log.Logger().Debugf("ignoring remote chart %s release %s", fullChartName, release.Name)
			continue
		}

		// lets not try resolve repository / versions for local charts
		if prefix != "." && prefix != ".." {
			repository, err = helmfiles.AddRepository([]*state.HelmState{helmState}, prefix, "", o.prefixes)
			if err != nil {
				return fmt.Errorf("failed to add repository for release %s: %w", release.Name, err)
			}

			// lets look for an override version label
			if stringhelpers.StringArrayIndex(ignoreRepositories, repository) < 0 && !IsLabelValue(&release, helmhelpers.VersionLabel, helmhelpers.LockLabelValue) {
				err = o.updateRelease(helmState, prefix, &release, fullChartName, repository, helmfile)
				if err != nil {
					return err
				}
				// If release.Chart has changed update related variables
				if fullChartName != release.Chart {
					fullChartName = release.Chart
					parts = strings.Split(fullChartName, "/")
					chartName = release.Chart
					if len(parts) > 1 {
						prefix = parts[0]
						chartName = parts[1]
					}
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
		if !IsLabelValue(&release, helmhelpers.ValuesLabel, helmhelpers.LockLabelValue) {
			found, err := o.updateValues(helmfile, filepath.Join(prefix, release.Name), &release)
			if err != nil {
				return errors.Wrapf(err, "failed to add values")
			}
			if !found {
				// next try the full chart name
				// ToDo: use this found value
				found, err = o.updateValues(helmfile, fullChartName, &release) //nolint:ineffassign,staticcheck
				if err != nil {
					return errors.Wrapf(err, "failed to add values")
				}
			}

			// lets try discover any local files
			found = false
			for _, releaseName := range releaseNames {
				for _, valueFileName := range valueFileNames {
					path := filepath.Join(helmfile.RelativePathToRoot, "values", releaseName, valueFileName)
					appValuesFile := filepath.Join(o.Dir, path)
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

			if helmfile.RelativePathToRoot != "" {
				if IsLabelValue(&release, helmhelpers.ValuesLabel, helmhelpers.NoRequirementsLabelValue) {
					for i, v := range release.Values {
						if v == reqvalues.RequirementsValuesFileName {
							release.Values = append(release.Values[:i], release.Values[i+1:]...)
						}
					}
				} else {
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
			}
		}
		helmState.Releases[i] = release
	}

	return nil
}

func (o *Options) ensureEnvironment(helmState *state.HelmState, helmfile helmfiles.Helmfile) {
	if helmfile.RelativePathToRoot != "" {
		// ensure we have added the jx-values.yaml file in the envirionment
		if helmState.Environments == nil {
			helmState.Environments = map[string]state.EnvironmentSpec{}
		}

		envSpec := helmState.Environments["default"]
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
}

func (o *Options) updateRelease(helmState *state.HelmState, prefix string, release *state.ReleaseSpec, fullChartName, repository string, helmfile helmfiles.Helmfile) error {
	// first try and match using the prefix and release name as we might have a version stream folder that uses helm alias
	versionProperties, err := o.Options.Resolver.StableVersion(versionstream.KindChart, prefix+"/"+release.Name)
	if err != nil {
		return errors.Wrapf(err, "failed to find version number for chart %s", release.Name)
	}
	// let's fall back to using the full chart name
	if versionProperties.Missing() {
		versionProperties, err = o.Options.Resolver.StableVersion(versionstream.KindChart, fullChartName)
		if err != nil {
			return errors.Wrapf(err, "failed to find version number for chart %s", fullChartName)
		}
	}
	if (versionProperties.ReplacementChart != "" || versionProperties.ReplacementChartPrefix != "") &&
		(release.Version == "" || o.UpdateMode) {
		if versionProperties.ReplacementChart != "" {
			release.Name = versionProperties.ReplacementChart
		}
		if versionProperties.ReplacementChartPrefix != "" {
			prefix = versionProperties.ReplacementChartPrefix
			newChart := fmt.Sprintf("%s/%s", prefix, release.Name)
			// Checking that replacement chart doesn't already exist in helmfile
			for i := range helmState.Releases {
				existingRelease := helmState.Releases[i]
				if existingRelease.Chart == newChart {
					log.Logger().Warningf("Can't replace %s with %s since %s already exist in helmfile. You should probably remove %s from %s yourself.", release.Chart, newChart, newChart, release.Chart, helmfile.Filepath)
					return nil
				}
			}
			release.Chart = newChart

			// let's make sure we have the repository
			found := false
			for k := range helmState.Repositories {
				repo := helmState.Repositories[k]
				if repo.Name == versionProperties.ReplacementChartPrefix {
					found = true
					break
				}
			}
			if !found {
				repository, err = versionstreamer.MatchRepositoryPrefix(o.prefixes, versionProperties.ReplacementChartPrefix)
				oci := strings.HasPrefix(repository, "oci://")
				if oci {
					repository = repository[len("oci://"):]
				}
				if err != nil {
					return err
				}
				helmState.Repositories = append(helmState.Repositories, state.RepositorySpec{
					Name: versionProperties.ReplacementChartPrefix,
					URL:  repository,
					OCI:  oci,
				})
			}
		}
		log.Logger().Debugf("replacing chart %s with %s", fullChartName, release.Chart)
		return o.updateRelease(helmState, prefix, release, release.Chart, repository, helmfile)
	}

	// Adding default labels
	if versionProperties.Labels != nil && release.Labels == nil {
		release.Labels = make(map[string]string)
	}
	for k, v := range versionProperties.Labels {
		release.Labels[k] = v
	}

	version := versionProperties.Version

	if release.Version == "" && version == "" {
		log.Logger().Debugf("could not find version for chart %s so using latest found in helm repository %s", fullChartName, repository)
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
		log.Logger().Debugf("resolved chart %s version %s", fullChartName, version)
	}

	if release.Namespace == "" && helmState.OverrideNamespace == "" && versionProperties.Namespace != "" {
		release.Namespace = versionProperties.Namespace
	}
	return nil
}

// IsLabelValue returns true if the release is labelled with the given label with a value
func IsLabelValue(release *state.ReleaseSpec, label, value string) bool {
	answer := false
	if release.Labels != nil {
		lockVersionValue := strings.TrimSpace(release.Labels[label])
		if lockVersionValue == value {
			answer = true
		}
	}
	return answer
}

func (o *Options) updateValues(helmfile helmfiles.Helmfile, name string, release *state.ReleaseSpec) (bool, error) {
	// Prune any reference to value file in version stream that has been removed
	release.Values = slices.DeleteFunc(release.Values, func(value any) bool {
		if valueFileName, ok := value.(string); ok {
			relativeValueFile, inVersionStream := strings.CutPrefix(
				valueFileName,
				filepath.Join(helmfile.RelativePathToRoot, versionStreamDir),
			)
			if inVersionStream {
				versionStreamValuesFile := filepath.Join(o.Resolver.VersionsDir, relativeValueFile)

				exists, err := files.FileExists(versionStreamValuesFile)
				if err != nil {
					log.Logger().Warnf("failed to check if version stream values file exists %s", versionStreamValuesFile)
					return false
				}
				return !exists
			}
		}
		return false
	})
	// Add any value file in version stream for chart
	found := false
	for _, valueFileName := range valueFileNames {
		versionStreamValuesFile := filepath.Join(o.Resolver.VersionsDir, "charts", name, valueFileName)
		exists, err := files.FileExists(versionStreamValuesFile)
		if err != nil {
			return false, errors.Wrapf(err, "failed to check if version stream values file exists %s", versionStreamValuesFile)
		}
		if exists {
			path := filepath.Join(helmfile.RelativePathToRoot, versionStreamDir, "charts", name, valueFileName)
			if !valuesContains(release.Values, path) {
				release.Values = append(release.Values, path)
			}
			found = true
			break
		}
	}
	return found, nil
}

// HasHelmfile returns true if there is a helmfile
func (o *Options) HasHelmfile() (bool, error) {
	if o.Helmfile == "" {
		o.Helmfile = filepath.Join(o.Dir, "helmfile.yaml")
	} else {
		o.Helmfile = filepath.Join(o.Dir, o.Helmfile)
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

func (o *Options) GitCommit(outDir, commitMessage string) error {
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
	if helmstate.Releases == nil {
		return nil
	}
	err := o.migrateQuickstartsFile()
	if err != nil {
		return errors.Wrapf(err, "failed to migrate quickstarts file")
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
		case "https://comcast.github.io/kuberhealthy/helm-repos":
			repo.URL = "https://kuberhealthy.github.io/kuberhealthy/helm-repos"
		case "https://godaddy.github.io/kubernetes-external-secrets":
			repo.URL = "https://external-secrets.github.io/kubernetes-external-secrets"
		case "https://chrismellard.github.io/kubernetes-external-secrets":
			repo.URL = "https://external-secrets.github.io/kubernetes-external-secrets"
		case "https://storage.googleapis.com/jenkinsxio/charts":
			if repo.Name == "external-secrets" {
				repo.URL = "https://external-secrets.github.io/kubernetes-external-secrets"
			}
		case "http://chartmuseum.jenkins-x.io":
			repo.URL = "https://storage.googleapis.com/chartmuseum.jenkins-x.io"
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
			for k := range helmstate.Repositories {
				repo := helmstate.Repositories[k]
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

	addedJxgh := false

	// lets replace the old jx3 charts
	for i := range helmstate.Releases {
		release := &helmstate.Releases[i]
		names := strings.SplitN(release.Chart, "/", 2)
		if len(names) == 2 && names[0] == "jx3" {
			chartName := names[1]
			name := release.Name
			release.Chart = "jxgh/" + chartName
			addedJxgh = true

			for j := range release.Values {
				v := release.Values[j]
				s, ok := v.(string)
				if !ok {
					continue
				}
				// lets switch invalid paths to the one inside a chart repo folder
				if s == fmt.Sprintf("%s/charts/jx3/%s/values.yaml.gotmpl", versionStreamPath, chartName) {
					release.Values[j] = fmt.Sprintf("%s/charts/jxgh/%s/values.yaml.gotmpl", versionStreamPath, chartName)
					break
				}
				if s == fmt.Sprintf("%s/charts/jx3/%s/values.yaml.gotmpl", versionStreamPath, name) {
					release.Values[j] = fmt.Sprintf("%s/charts/jxgh/%s/values.yaml.gotmpl", versionStreamPath, name)
					break
				}
			}
		}
	}

	// lets replace the old jenkins-x charts
	for _, chartName := range []string{"jxboot-helmfile-resources", "bucketrepo", "nexus"} {
		for i := range helmstate.Releases {
			release := &helmstate.Releases[i]
			if release.Chart == "jenkins-x/"+chartName {
				release.Chart = "jxgh/" + chartName
				addedJxgh = true

				for j := range release.Values {
					v := release.Values[j]
					s, ok := v.(string)
					// lets switch invalid paths to the one inside a chart repo folder
					if ok && s == fmt.Sprintf("%s/charts/jenkins-x/%s/values.yaml.gotmpl", versionStreamPath, chartName) {
						release.Values[j] = fmt.Sprintf("%s/charts/jxgh/%s/values.yaml.gotmpl", versionStreamPath, chartName)
						break
					}
				}
			}
		}
	}

	// lets replace the old lighthouse chart repository location if its being used
	for i := range helmstate.Releases {
		release := &helmstate.Releases[i]
		if release.Chart == "jenkins-x/lighthouse" {
			release.Chart = "jxgh/lighthouse"
			addedJxgh = true

			for i := range release.Values {
				v := release.Values[i]
				s, ok := v.(string)
				// lets switch invalid paths to the one inside a chart repo folder
				if ok && s == fmt.Sprintf("%s/charts/jenkins-x/lighthouse/values.yaml.gotmpl", versionStreamPath) {
					release.Values[i] = fmt.Sprintf("%s/charts/jxgh/lighthouse/values.yaml.gotmpl", versionStreamPath)
					break
				}
			}
		}
	}

	// lets check for terraform installed vault
	if requirements.SecretStorage == jxcore.SecretStorageTypeVault && requirements.TerraformVault {
		for i := range helmstate.Releases {
			release := &helmstate.Releases[i]
			if release.Chart == "jxgh/vault-instance" || release.Chart == "banzaicloud-stable/vault-operator" {
				log.Logger().Infof("Terraform installed detected and Vault chart %s still present in helmfile. Please migrate secrets as necessary and remove this chart from your helmfile", release.Chart)
			}
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
			for k := range helmstate.Repositories {
				repo := helmstate.Repositories[k]
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
			for k := range helmstate.Repositories {
				repo := helmstate.Repositories[k]
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
			if release.Chart == "jxgh/local-external-secrets" {
				found = true
				break
			}
		}
		if !found {
			release := state.ReleaseSpec{
				Chart: "jxgh/local-external-secrets",
			}
			addedJxgh = true
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
				release.Chart = "jxgh/" + name
				addedJxgh = true
				if name == "jenkins-x-crds" {
					release.Values = []interface{}{fmt.Sprintf("%s/charts/jxgh/jenkins-x-crds/values.yaml.gotmpl", versionStreamPath)}
				}
				o.updateVersionFromVersionStream(release)
				break
			}
		}
	}

	// lets ensure we have the jx-build-controller installed
	if helmstate.OverrideNamespace == "jx" && isDevCluster(helmstate) {
		found := false
		for i := range helmstate.Releases {
			release := &helmstate.Releases[i]
			if release.Chart == "jxgh/jx-build-controller" {
				found = true
				break
			}
		}
		if !found && helmstate.OverrideNamespace == "jx" {
			helmstate.Releases = append(helmstate.Releases, state.ReleaseSpec{
				Chart: "jxgh/jx-build-controller",
			})
			addedJxgh = true
		}
	}

	if addedJxgh {
		// lets make sure we have a jxgh repository
		found := false
		for k := range helmstate.Repositories {
			repo := helmstate.Repositories[k]
			if repo.Name == "jxgh" {
				found = true
				break
			}
		}
		if !found {
			helmstate.Repositories = append(helmstate.Repositories, state.RepositorySpec{
				Name: "jxgh",
				URL:  "https://jenkins-x-charts.github.io/repo",
			})
		}
	}

	if o.AddEnvironmentPipelines {
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
	}
	return nil
}

// removeRedundantRepositories removes any repositories from a state.HelmState that are not referenced by any releases
func removeRedundantRepositories(helmstate *state.HelmState) {
	requiredRepositories := make(map[string]bool)
	for i := range helmstate.Releases {
		repoName := strings.SplitN(helmstate.Releases[i].Chart, "/", 2)[0]
		requiredRepositories[repoName] = true
	}

	var cleanedRepositories []state.RepositorySpec
	for i := range helmstate.Repositories {
		if requiredRepositories[helmstate.Repositories[i].Name] {
			cleanedRepositories = append(cleanedRepositories, helmstate.Repositories[i])
		}
	}
	helmstate.Repositories = cleanedRepositories
}

func isDevCluster(helmState *state.HelmState) bool {
	for k := range helmState.Releases {
		release := helmState.Releases[k]
		_, local := helmfiles.SpitChartName(release.Chart)
		if local == "jxboot-helmfile-resources" || local == "lighthouse" {
			return true
		}
	}
	return false
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

func (o *Options) upgradePipelineCatalog() error {
	pc, path, err := pipelinecatalogs.LoadPipelineCatalogs(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to load pipeline catalogs")
	}

	modified := false
	for i := range pc.Spec.Repositories {
		repo := &pc.Spec.Repositories[i]
		gitURL := repo.GitURL
		if gitURL != "" {
			version, err := o.Options.Resolver.ResolveGitVersion(gitURL)
			if err != nil {
				return errors.Wrapf(err, "failed to find stable version of pipeline catalog %s", gitURL)
			}
			if version != "" && version != repo.GitRef {
				modified = true
				repo.GitRef = version
				log.Logger().Infof("updated version of pipeline catalog %s to %s", info(gitURL), info(version))
			}
		}
	}
	if !modified {
		return nil
	}

	err = yamls.SaveFile(pc, path)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", path)
	}
	log.Logger().Infof("modified %s", info(path))
	return nil
}

func (o *Options) migrateQuickstartsFile() error {
	qs, path, err := quickstarthelpers.LoadQuickstarts(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to load quickstarts")
	}

	modified := false
	for i := range qs.Spec.Imports {
		ip := &qs.Spec.Imports[i]
		if strings.HasSuffix(ip.File, ".yml") {
			ip.File = strings.TrimSuffix(ip.File, ".yml") + ".yaml"
			modified = true
		}
	}
	if !modified {
		return nil
	}

	err = yamls.SaveFile(qs, path)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", path)
	}
	log.Logger().Infof("patched %s to use correct .yaml extension", info(path))
	return nil
}
