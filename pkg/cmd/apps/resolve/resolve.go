package resolve

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-apps/pkg/helmfile"
	"github.com/jenkins-x/jx-gitops/pkg/jxtmpl/reqvalues"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/pkg/versionstream/versionstreamrepo"

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
	AppsCfg                    *jxapps.AppConfig
	AppsCfgFile                string
	VersionsDir                string
	RequirementsValuesFileName string
	Resolver                   *versionstream.VersionResolver
	Requirements               *config.RequirementsConfig
}

// NewCmdJxAppsTemplate creates a command object for the command
func NewCmdJxAppsResolve() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "resolve",
		Short:   "Resolves the jx-apps.yml from the version stream to specify versions and helm values",
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
	cmd.Flags().StringVarP(&o.VersionStreamDir, "version-stream-dir", "", "", "optional directory that contains a version stream")
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
	appsCfg, appsCfgFile, err := jxapps.LoadAppConfig(o.Dir)
	if err != nil {
		return errors.Wrap(err, "failed to load jx-apps.yml")
	}

	o.Results.AppsCfg = appsCfg
	o.Results.AppsCfgFile = appsCfgFile

	versionsDir := o.VersionStreamDir
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
		if o.VersionStreamURL == "" {
			return errors.Errorf("Missing option:  --%s ", termcolor.ColorInfo("url"))
		}

		var err error
		o.VersionStreamDir, err = ioutil.TempDir("", "jx-version-stream-")
		if err != nil {
			return errors.Wrap(err, "failed to create temp dir")
		}

		versionsDir, _, err = versionstreamrepo.CloneJXVersionsRepoToDir(o.Dir, o.VersionStreamURL, o.VersionStreamRef, nil, o.Git(), true, false, files.GetIOFileHandles(o.IOFileHandles))
		if err != nil {
			return errors.Wrapf(err, "failed to clone version stream to %s", o.Dir)
		}
	}
	if o.GitCommitMessage == "" {
		o.GitCommitMessage = "chore: resolved applications from the version stream"
	}

	if o.Results.Resolver == nil {
		o.Results.Resolver = &versionstream.VersionResolver{
			VersionsDir: versionsDir,
		}
	}
	o.prefixes, err = o.Results.Resolver.GetRepositoryPrefixes()
	if err != nil {
		return errors.Wrapf(err, "failed to load repository prefixes at %s", versionsDir)
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
	appsCfg := o.Results.AppsCfg
	if appsCfg == nil {
		return errors.Errorf("failed to load the jx-apps.yml")
	}
	resolver := o.Results.Resolver
	if resolver == nil {
		return errors.Errorf("failed to create the VersionResolver")
	}
	versionsDir := resolver.VersionsDir
	appsCfgFile := o.Results.AppsCfgFile
	appsCfgDir := filepath.Dir(appsCfgFile)

	requirementsValuesFiles := o.Results.RequirementsValuesFileName
	if requirementsValuesFiles != "" {
		if stringhelpers.StringArrayIndex(appsCfg.Values, requirementsValuesFiles) < 0 {
			appsCfg.Values = append(appsCfg.Values, requirementsValuesFiles)
		}
	}
	count := 0
	for i, app := range appsCfg.Apps {
		repository := app.Repository
		fullChartName := app.Name
		parts := strings.Split(app.Name, "/")
		if len(parts) != 2 {
			return errors.Wrapf(err, "failed to find prefix in the form prefix/name from app name %s", app.Name)
		}
		prefix := parts[0]
		chartName := parts[1]

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
				appsCfg.Repositories = append(appsCfg.Repositories, helmfile.RepositorySpec{
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

		if app.Namespace == "" && appsCfg.DefaultNamespace != "" {
			app.Namespace = appsCfg.DefaultNamespace
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

		// TODO where to put the jx requirements values file? common area?
		// ho.ValuesFiles = append(ho.ValuesFiles, jxReqValuesFileName)

		for _, valueFileName := range valueFileNames {
			versionStreamPath := filepath.Join("apps", prefix, chartName, valueFileName)
			appValuesFile := filepath.Join(versionsDir, versionStreamPath)
			exists, err := files.FileExists(appValuesFile)
			if err != nil {
				return errors.Wrapf(err, "failed to check if app values file exists %s", appValuesFile)
			}
			if exists {
				chartAppsParentDir := filepath.Join("versionStream", "apps", prefix)
				chartAppsDir := filepath.Join(chartAppsParentDir, chartName)
				path := filepath.Join(chartAppsDir, valueFileName)
				if stringhelpers.StringArrayIndex(app.Values, path) < 0 {
					// lets make sure the parent dir exists
					d := filepath.Join(o.Dir, chartAppsParentDir)
					err = os.MkdirAll(d, files.DefaultDirWritePermissions)
					if err != nil {
						return errors.Wrapf(err, "failed to create dir %s", d)
					}
					log.Logger().Infof("created dir %s", d)

					if o.VersionStreamURL == "" {
						return errors.Errorf("cannot use kpt to get the helm versions file %s from the version stream as no version stream git URL provided", path)
					}

					// lets use kpt to copy the values file from the version stream locally
					c := &cmdrunner.Command{
						Dir:  o.Dir,
						Name: "kpt",
						Args: []string{
							"pkg",
							"get",
							fmt.Sprintf("%s/%s@%s", o.VersionStreamURL, versionStreamPath, o.VersionStreamRef),
							chartAppsDir,
						},
					}
					_, err = o.CommandRunner(c)
					if err != nil {
						return errors.Wrapf(err, "failed to run command %s", c.CLI())
					}
					app.Values = append(app.Values, path)
				}
			}
		}

		releaseNames := []string{chartName}
		if app.Alias != "" && app.Alias != chartName {
			releaseNames = []string{app.Alias, chartName}
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
					if stringhelpers.StringArrayIndex(app.Values, path) < 0 {
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

		appsCfg.Apps[i] = app
	}
	err = appsCfg.SaveConfig(appsCfgFile)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", appsCfgFile)
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
