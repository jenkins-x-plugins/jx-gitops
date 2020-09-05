package resolve

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/helmhelpers"
	"github.com/jenkins-x/jx-gitops/pkg/jxtmpl/reqvalues"
	"github.com/jenkins-x/jx-gitops/pkg/versionstreamer"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/pkg/yaml2s"
	"github.com/roboll/helmfile/pkg/state"

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
	BatchMode        bool
	UpdateMode       bool
	DoGitCommit      bool
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
	versionsDir := resolver.VersionsDir
	appsCfgDir := o.Dir

	log.Logger().Infof("resolving versions and values files from the version stream %s ref %s in dir %s", o.VersionStreamURL, o.VersionStreamRef, o.VersionStreamDir)

	helmState := o.Results.HelmState

	err = helmhelpers.AddHelmRepositories(helmState, o.CommandRunner)
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
			versionProperties, err := resolver.StableVersion(versionstream.KindChart, fullChartName)
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
			versionStreamValuesFile := filepath.Join(versionsDir, "charts", prefix, chartName, valueFileName)
			exists, err := files.FileExists(versionStreamValuesFile)
			if err != nil {
				return errors.Wrapf(err, "failed to check if version stream values file exists %s", versionStreamValuesFile)
			}
			if exists {
				path := filepath.Join("versionStream", "charts", prefix, chartName, valueFileName)
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
