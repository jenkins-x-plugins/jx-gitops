package add

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/versionstreamer"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yaml2s"
	"github.com/roboll/helmfile/pkg/state"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/versionstream"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Adds a chart to the local 'helmfile.yaml' file
`)

	cmdExample = templates.Examples(`
		# adds a chart using the currently known repositories in the verison stream or helmfile.yaml
		%s helmfile add --chart somerepo/mychart

		# adds a chart using a new repository URL with a custom version and namespace
		%s helmfile add --chart somerepo/mychart --repository https://acme.com/myrepo --namespace foo --version 1.2.3
	`)
)

// Options the options for the command
type Options struct {
	versionstreamer.Options
	Namespace        string
	GitCommitMessage string
	Helmfile         string
	Chart            string
	Repository       string
	Version          string
	ReleaseName      string
	Values           []string
	BatchMode        bool
	DoGitCommit      bool
	Gitter           gitclient.Interface
	prefixes         *versionstream.RepositoryPrefixes
	Results          Results
}

type Results struct {
	HelmState                  state.HelmState
	RequirementsValuesFileName string
}

// NewCmdHelmfileAdd creates a command object for the command
func NewCmdHelmfileAdd() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "add",
		Short:   "Adds a chart to the local 'helmfile.yaml' file",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.Options.AddFlags(cmd)

	cmd.Flags().StringVarP(&o.Helmfile, "helmfile", "", "", "the helmfile to resolve. If not specified defaults to 'helmfile.yaml' in the dir")
	cmd.Flags().StringVarP(&o.GitCommitMessage, "commit-message", "", "chore: generated kubernetes resources from helm chart", "the git commit message used")

	// chart flags
	cmd.Flags().StringVarP(&o.Chart, "chart", "c", "", "the name of the helm chart to add")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "jx", "the namespace to install the chart")
	cmd.Flags().StringVarP(&o.ReleaseName, "name", "", "", "the name of the helm release")
	cmd.Flags().StringVarP(&o.Repository, "repository", "r", "", "the helm chart repository URL of the chart")
	cmd.Flags().StringVarP(&o.Version, "version", "v", "", "the version of the helm chart. If not specified the versionStream will be checked otherwise the latest version is used")
	cmd.Flags().StringArrayVarP(&o.Values, "values", "", nil, "the values files to add to the chart")

	// git commit stuff....
	cmd.Flags().BoolVarP(&o.DoGitCommit, "git-commit", "", false, "if set then the template command will git commit the modified helmfile.yaml files")

	return cmd, o
}

// Validate validates the options and populates any missing values
func (o *Options) Validate() error {
	err := o.Options.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate")
	}

	if o.Chart == "" {
		return options.MissingOption("chart")
	}
	if o.Helmfile == "" {
		o.Helmfile = filepath.Join(o.Dir, "helmfiles", o.Namespace, "helmfile.yaml")
	}

	o.prefixes, err = o.Options.Resolver.GetRepositoryPrefixes()
	if err != nil {
		return errors.Wrapf(err, "failed to load repository prefixes at %s", o.VersionStreamDir)
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

	helmState := o.Results.HelmState

	modified := false
	found := false

	parts := strings.Split(o.Chart, "/")
	prefix := ""
	if len(parts) > 1 {
		prefix = parts[0]
	}
	repository := o.Repository

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
					return errors.Errorf("release %s has prefix %s for repository URL %s which is also mapped to prefix %s", o.Chart, prefix, r.URL, r.Name)
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

	for i := range helmState.Releases {
		release := &helmState.Releases[i]
		if release.Chart == o.Chart && release.Name == o.ReleaseName {
			found = true
			if release.Namespace != "" && release.Namespace != o.Namespace {
				release.Namespace = o.Namespace
				modified = true
			}

			// lets add any missing values
			for _, v := range o.Values {
				foundValue := false
				for j := range release.Values {
					if release.Values[j] == v {
						foundValue = true
						break
					}
				}
				if !foundValue {
					release.Values = append(release.Values, v)
					modified = true
				}
			}
			break
		}
	}
	if !found {
		release := state.ReleaseSpec{
			Chart:     o.Chart,
			Version:   o.Version,
			Name:      o.ReleaseName,
			Namespace: o.Namespace,
		}
		for _, v := range o.Values {
			release.Values = append(release.Values, v)
		}
		helmState.Releases = append(helmState.Releases, release)
		modified = true
	}
	if !modified {
		log.Logger().Debugf("no changes were made to file %s", o.Helmfile)
		return nil
	}

	dir := filepath.Dir(o.Helmfile)
	err = os.MkdirAll(dir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create dir %s", dir)
	}

	err = yaml2s.SaveFile(helmState, o.Helmfile)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", o.Helmfile)
	}

	err = o.ensureHelmfileInRootHelmfile(o.Helmfile)
	if err != nil {
		return errors.Wrapf(err, "failed to reference the helmfile %s in the root helmfile", o.Helmfile)
	}

	_, err = o.Git().Command(o.Dir, "add", "*")
	if err != nil {
		return errors.Wrapf(err, "failed to add helmfile changes to git in dir %s", o.Dir)
	}

	if !o.DoGitCommit {
		return nil
	}
	log.Logger().Infof("committing changes: %s", o.GitCommitMessage)
	err = o.GitCommit(o.Dir, o.GitCommitMessage)
	if err != nil {
		return errors.Wrapf(err, "failed to commit changes")
	}
	return nil
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
	err := gitclient.CommitIfChanges(gitter, outDir, commitMessage)
	if err != nil {
		return errors.Wrapf(err, "failed to commit changes to git in dir %s", outDir)
	}
	return nil
}

func (o *Options) ensureHelmfileInRootHelmfile(path string) error {
	rel, err := filepath.Rel(o.Dir, path)
	if err != nil {
		return errors.Wrapf(err, "failed to get relative path of %s to %s", path, o.Dir)
	}

	root := filepath.Join(o.Dir, "helmfile.yaml")
	rootState := &state.HelmState{}
	err = yaml2s.LoadFile(root, rootState)
	if err != nil {
		return errors.Wrapf(err, "failed to load root helmfile %s", root)
	}

	for _, hf := range rootState.Helmfiles {
		if hf.Path == rel {
			return nil
		}
	}
	rootState.Helmfiles = append(rootState.Helmfiles, state.SubHelmfileSpec{
		Path: rel,
	})

	sort.Slice(rootState.Helmfiles, func(i, j int) bool {
		h1 := rootState.Helmfiles[i]
		h2 := rootState.Helmfiles[j]
		return h1.Path < h2.Path
	})

	err = yaml2s.SaveFile(rootState, root)
	if err != nil {
		return errors.Wrapf(err, "failed to save %s", root)
	}
	log.Logger().Infof("added new child helmfile %s to root file %s", info(rel), info(root))
	return nil
}
