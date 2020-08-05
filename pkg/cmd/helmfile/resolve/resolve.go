package resolve

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/jxtmpl/reqvalues"
	"github.com/jenkins-x/jx-gitops/pkg/yamlvs"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/pkg/versionstream/versionstreamrepo"
	"github.com/roboll/helmfile/pkg/state"

	"github.com/jenkins-x/jx-apps/pkg/jxapps"

	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/versionstream"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Resolves the jx-apps.yml from the version stream to specify versions and helm values
`)

	cmdExample = templates.Examples(`
		# resolves the versions and values in the jx-apps.yml
		%s step apps resolve
	`)

	valueFileNames = []string{"values.yaml.gotmpl", "values.yaml"}
)

// Options the options for the command
type Options struct {
	Namespace        string
	GitCommitMessage string
	Dir              string
	Helmfile         string
	VersionStreamDir string
	VersionStreamURL string
	VersionStreamRef string
	BatchMode        bool
	UpdateMode       bool
	DoGitCommit      bool
	IOFileHandles    *files.IOFileHandles
	Gitter           gitclient.Interface
	CommandRunner    cmdrunner.CommandRunner
	prefixes         *versionstream.RepositoryPrefixes
	Results          Results
}

type Results struct {
	AppsCfg                    state.HelmState
	VersionsDir                string
	RequirementsValuesFileName string
	Resolver                   *versionstream.VersionResolver
	Requirements               *config.RequirementsConfig
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
	o.AddFlags(cmd, "")
	return cmd, o
}

func (o *Options) AddFlags(cmd *cobra.Command, prefix string) {
	cmd.Flags().BoolVarP(&o.UpdateMode, "update", "", false, "updates versions from the version stream if they have changed")
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory that contains the jx-apps.yml")
	cmd.Flags().StringVarP(&o.Helmfile, "helmfile", "", "", "the helmfile to resolve. If not specified defaults to 'helmfile.yaml' in the dir")
	cmd.Flags().StringVarP(&o.VersionStreamDir, "version-stream-dir", "", "", "the directory for the version stream")
	cmd.Flags().StringVarP(&o.GitCommitMessage, prefix+"commit-message", "", "chore: generated kubernetes resources from helm chart", "the git commit message used")
	cmd.Flags().StringVarP(&o.VersionStreamURL, "url", "n", "", "the git clone URL of the version stream. If not specified it defaults to the value in the jx-requirements.yml")
	cmd.Flags().StringVarP(&o.VersionStreamRef, "ref", "c", "", "the git ref (branch, tag, revision) of the version stream to git clone. If not specified it defaults to the value in the jx-requirements.yml")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", "jx", "the default namespace if none is specified in the jx-apps.yml or jx-requirements.yml")

	// git commit stuff....
	cmd.Flags().BoolVarP(&o.DoGitCommit, prefix+"git-commit", "", false, "if set then the template command will git commit the modified jx-apps.yml files")
}

// Validate validates the options and populates any missing values
func (o *Options) Validate() error {
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
	}

	if o.Helmfile == "" {
		o.Helmfile = filepath.Join(o.Dir, "helmfile.yaml")
	}

	err := yamlvs.LoadFile(o.Helmfile, &o.Results.AppsCfg)
	if err != nil {
		return errors.Wrapf(err, "failed to load helmfile %s", o.Helmfile)
	}

	requirements, _, err := config.LoadRequirementsConfig(o.Dir, false)
	if err != nil {
		return errors.Wrapf(err, "failed to load jx-requirements.yml")
	}
	o.Results.Requirements = requirements
	if o.VersionStreamURL == "" {
		o.VersionStreamURL = requirements.VersionStream.URL
		if o.VersionStreamURL == "" {
			o.VersionStreamURL = requirements.VersionStream.URL
		}
	}
	if o.VersionStreamRef == "" {
		o.VersionStreamRef = requirements.VersionStream.Ref
		if o.VersionStreamRef == "" {
			o.VersionStreamRef = "master"
		}
	}
	if o.VersionStreamDir == "" {
		o.VersionStreamDir = filepath.Join(o.Dir, "versionStream")
	}

	err = o.ResolveVersionStream()
	if err != nil {
		return errors.Wrapf(err, "failed to resolve the version stream")
	}

	if o.VersionStreamDir == "" {
		if o.VersionStreamURL == "" {
			return errors.Errorf("Missing option:  --%s ", termcolor.ColorInfo("url"))
		}

		var err error
		o.VersionStreamDir, err = ioutil.TempDir("", "jx-version-stream-")
		if err != nil {
			return errors.Wrap(err, "failed to create temp dir")
		}

		o.VersionStreamDir, _, err = versionstreamrepo.CloneJXVersionsRepoToDir(o.VersionStreamDir, o.VersionStreamURL, o.VersionStreamRef, nil, o.Git(), true, false, files.GetIOFileHandles(o.IOFileHandles))
		if err != nil {
			return errors.Wrapf(err, "failed to clone version stream to %s", o.Dir)
		}
	}
	if o.GitCommitMessage == "" {
		o.GitCommitMessage = "chore: resolved applications from the version stream"
	}

	if o.Results.Resolver == nil {
		o.Results.Resolver = &versionstream.VersionResolver{
			VersionsDir: o.VersionStreamDir,
		}
	}
	o.prefixes, err = o.Results.Resolver.GetRepositoryPrefixes()
	if err != nil {
		return errors.Wrapf(err, "failed to load repository prefixes at %s", o.VersionStreamDir)
	}

	jxReqValuesFileName := filepath.Join(o.Dir, reqvalues.RequirementsValuesFileName)
	o.Results.RequirementsValuesFileName = reqvalues.RequirementsValuesFileName
	err = reqvalues.SaveRequirementsValuesFile(requirements, jxReqValuesFileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save tempo file for jx requirements values file %s", jxReqValuesFileName)
	}
	return nil
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to ")
	}

	resolver := o.Results.Resolver
	if resolver == nil {
		return errors.Errorf("failed to create the VersionResolver")
	}
	versionsDir := resolver.VersionsDir
	appsCfgDir := o.Dir

	log.Logger().Infof("resolving versions and values files from the version stream %s ref %s in dir %s", o.VersionStreamURL, o.VersionStreamRef, o.VersionStreamDir)

	appsCfg := o.Results.AppsCfg

	/*
		TODO lazily create environments file?
		requirementsValuesFiles := o.Results.RequirementsValuesFileName
		if requirementsValuesFiles != "" {
			if stringhelpers.StringArrayIndex(appsCfg.Values, requirementsValuesFiles) < 0 {
				appsCfg.Values = append(appsCfg.Values, requirementsValuesFiles)
			}
		}

	*/
	count := 0
	for i, app := range appsCfg.Releases {
		// TODO
		//repository := app.Repository
		repository := ""
		fullChartName := app.Chart
		parts := strings.Split(fullChartName, "/")
		prefix := ""
		chartName := app.Chart
		if len(parts) > 1 {
			prefix = parts[0]
			chartName = parts[1]
		}
		if app.Name == "" {
			app.Name = chartName
		}

		// lets resolve the chart prefix from a local repository from the file or from a
		// prefix in the versions stream
		if repository == "" && prefix != "" {
			for _, r := range appsCfg.Repositories {
				if r.Name == prefix {
					repository = r.URL
				}
			}
		}
		if repository == "" && prefix != "" {
			repository, err = o.matchPrefix(prefix)
			if err != nil {
				return errors.Wrapf(err, "failed to match prefix %s with repositories from versionstream %s", prefix, o.VersionStreamURL)
			}
		}
		if repository == "" && prefix != "" {
			return errors.Wrapf(err, "failed to find repository URL, not defined in jx-apps.yml or versionstream %s", o.VersionStreamURL)
		}
		if repository != "" && prefix != "" {
			// lets ensure we've got a repository for this URL in the apps file
			found := false
			for _, r := range appsCfg.Repositories {
				if r.Name == prefix {
					if r.URL != repository {
						return errors.Errorf("app %s has prefix %s for repository URL %s which is also mapped to prefix %s", app.Name, prefix, r.URL, r.Name)
					}
					found = true
					break
				}
			}
			if !found {
				appsCfg.Repositories = append(appsCfg.Repositories, state.RepositorySpec{
					Name: prefix,
					URL:  repository,
				})
			}
		}
		version, err := resolver.StableVersionNumber(versionstream.KindChart, fullChartName)
		if err != nil {
			return errors.Wrapf(err, "failed to find version number for chart %s", fullChartName)
		}

		versionChanged := false
		if app.Version == "" {
			app.Version = version
			versionChanged = true
		} else if o.UpdateMode && app.Version != version {
			app.Version = version
			versionChanged = true
		}
		if versionChanged {
			log.Logger().Infof("resolved chart %s version %s", fullChartName, version)
		}

		defaultsDir := filepath.Join(versionsDir, string(versionstream.KindApp), fullChartName)
		defaults, _, err := jxapps.LoadAppDefaultsConfig(defaultsDir)
		if err != nil {
			return errors.Wrapf(err, "failed to load defaults from dir %s", defaultsDir)
		}

		if version == "" {
			log.Logger().Warnf("could not find version for chart %s so using latest found in helm repository %s", fullChartName, repository)
		}

		if app.Namespace == "" && defaults.Namespace != "" {
			app.Namespace = defaults.Namespace
		}

		if app.Namespace == "" && o.Results.Requirements != nil {
			app.Namespace = o.Results.Requirements.Cluster.Namespace
			if app.Namespace == "" {
				app.Namespace = o.Namespace
			}
		}

		for _, valueFileName := range valueFileNames {
			versionStreamPath := filepath.Join("apps", prefix, chartName)
			appValuesFile := filepath.Join(versionsDir, versionStreamPath, valueFileName)
			exists, err := files.FileExists(appValuesFile)
			if err != nil {
				return errors.Wrapf(err, "failed to check if app values file exists %s", appValuesFile)
			}
			if exists {
				path := filepath.Join("versionStream", "apps", prefix, chartName, valueFileName)
				if !valuesContains(app.Values, path) {
					app.Values = append(app.Values, path)
				}
			}
		}

		releaseNames := []string{chartName}
		if app.Name != "" && app.Name != chartName {
			releaseNames = []string{app.Name, chartName}
		}

		// lets try discover any local files
		found := false
		for _, releaseName := range releaseNames {
			for _, valueFileName := range valueFileNames {
				path := filepath.Join("apps", releaseName, valueFileName)
				appValuesFile := filepath.Join(appsCfgDir, path)
				exists, err := files.FileExists(appValuesFile)
				if err != nil {
					return errors.Wrapf(err, "failed to check if app values file exists %s", appValuesFile)
				}
				if exists {
					if !valuesContains(app.Values, path) {
						app.Values = append(app.Values, path)
					}
					found = true
					break
				}
			}
			if found {
				break
			}
		}

		appsCfg.Releases[i] = app
	}

	err = yamlvs.SaveFile(appsCfg, o.Helmfile)
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

func valuesContains(values []interface{}, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func (o *Options) matchPrefix(prefix string) (string, error) {
	if o.prefixes == nil {
		return "", errors.Errorf("no repository prefixes found in version stream")
	}
	// default to first URL
	repoURL := o.prefixes.URLsForPrefix(prefix)

	if repoURL == nil || len(repoURL) == 0 {
		return "", errors.Errorf("no matching repository for for prefix %s", prefix)
	}
	return repoURL[0], nil
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

// ResolveVersionStream verifies there is a valid version stream and if not resolves it via kpt
func (o *Options) ResolveVersionStream() error {
	chartsDir := filepath.Join(o.VersionStreamDir, "charts")
	exists, err := files.DirExists(chartsDir)
	if err != nil {
		return errors.Wrapf(err, "failed to check version stream dir exists %s", chartsDir)
	}
	if exists {
		return nil
	}
	versionStreamPath, err := filepath.Rel(o.Dir, o.VersionStreamDir)
	if err != nil {
		return errors.Wrapf(err, "failed to get relative path of version stream %s in %s", o.VersionStreamDir, o.Dir)
	}

	// lets use kpt to copy the values file from the version stream locally
	c := &cmdrunner.Command{
		Dir:  o.Dir,
		Name: "kpt",
		Args: []string{
			"pkg",
			"get",
			fmt.Sprintf("%s/%s@%s", o.VersionStreamURL, versionStreamPath, o.VersionStreamRef),
			o.Dir,
		},
	}
	_, err = o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve version stream %s ref %s using kpt", o.VersionStreamURL, o.VersionStreamRef)
	}
	return nil
}
