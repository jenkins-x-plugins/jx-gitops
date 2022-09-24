package structure

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/helmfile/helmfile/pkg/state"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yaml2s"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	HelmfileFolder = "helmfiles"
	helmfileName   = "helmfile.yaml"
)

var (
	cmdLong = templates.LongDesc(`
		Runs 'helmfile structure' on the helmfile in specified directory which will split in to multiple helmfiles based around namespace
`)

	cmdExample = templates.Examples(`
		# splits the helmfile.yaml into separate files for each namespace
		%s helmfile structure --dir /path/to/gitops/repo
	`)
)

// Options the options for the command
type Options struct {
	Dir      string
	Helmfile string
}

// NewCmdHelmfileTemplate creates a command object for the command
func NewCmdHelmfileStructure() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "structure",
		Short:   "Runs 'helmfile structure' on the helmfile in specified directory which will split in to multiple helmfiles based around namespace",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to run the commands inside")

	return cmd, o
}

// Validate validates the options and populates any missing values
func (o *Options) Validate() error {
	var err error
	if o.Helmfile == "" {
		o.Helmfile = filepath.Join(o.Dir, "helmfile.yaml")
	}
	exists, err := files.FileExists(o.Helmfile)
	if err != nil {
		return errors.Wrapf(err, "failed to check if file exists %s", o.Helmfile)
	}
	if !exists {
		return errors.Errorf("helmfile %s does not exist", o.Helmfile)
	}
	return nil
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate")
	}

	parentHelmState, _ := loadHelmfile(o.Helmfile)

	namespaceReleases := gatherNamespaceReleases(parentHelmState)

	configureHelmStatePaths(namespaceReleases)

	parentHelmState = configureParentHelmState(*parentHelmState, namespaceReleases)

	for ns, hs := range namespaceReleases {
		helmfile := getHelmfileAbsolute(o.Dir, ns)
		err = saveHelmState(helmfile, hs, false)
		if err != nil {
			return errors.Wrapf(err, "error saving helmfile %s", helmfile)
		}
	}

	err = saveHelmState(o.Helmfile, parentHelmState, true)
	if err != nil {
		return errors.Wrapf(err, "aborting save as file exists")
	}

	return nil
}

func getHelmfileRelative(namespace string) string {
	return filepath.Join(HelmfileFolder, namespace, helmfileName)
}

func getHelmfileAbsolute(workingDirectory, namespace string) string {
	return filepath.Join(workingDirectory, getHelmfileRelative(namespace))
}

func configureParentHelmState(helmState state.HelmState, nestedStates map[string]*state.HelmState) *state.HelmState { //nolint:gocritic
	hs := state.HelmState{
		FilePath:       helmState.FilePath,
		ReleaseSetSpec: helmState.ReleaseSetSpec,
		RenderedValues: helmState.RenderedValues,
	}
	hs.Releases = nil
	hs.Repositories = nil
	hs.Environments = nil
	if hs.Helmfiles == nil {
		hs.Helmfiles = []state.SubHelmfileSpec{}
	}

	keys := make([]string, 0, len(nestedStates))
	for k := range nestedStates {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, ns := range keys {
		hs.Helmfiles = append(hs.Helmfiles, state.SubHelmfileSpec{
			Path: getHelmfileRelative(ns),
		})
	}
	return &hs
}

func saveHelmState(filename string, helmState *state.HelmState, overwrite bool) error {
	if b, _ := files.FileExists(filename); b && !overwrite {
		return fmt.Errorf("helmfile already exists at %s, overwriting disabled and merging of helmfiles not currently supported", filename)
	}

	helmDir := filepath.Dir(filename)
	err := os.MkdirAll(helmDir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create directory %s", helmDir)
	}

	err = yaml2s.SaveFile(helmState, filename)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", filename)
	}
	return nil
}

func configureHelmStatePaths(releases map[string]*state.HelmState) {
	for j := range releases {
		hs := releases[j]
		for k := range hs.Releases {
			r := hs.Releases[k]
			for i, v := range r.Values {
				switch m := v.(type) { //nolint:gocritic
				// Explicit value strings are considered paths that need rewriting
				case string:
					r.Values[i] = filepath.Join("..", "..", m)
				}
			}
		}
		for _, env := range hs.Environments {
			for i, v := range env.Values {
				switch m := v.(type) { //nolint:gocritic
				// Explicit value strings are considered paths that need rewriting
				case string:
					env.Values[i] = filepath.Join("..", "..", m)
				}
			}
		}
	}
}

func loadHelmfile(file string) (*state.HelmState, error) {
	helmState := state.HelmState{}
	err := yaml2s.LoadFile(file, &helmState)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load helmfile %s", file)
	}
	return &helmState, nil
}

func gatherNamespaceReleases(helmstate *state.HelmState) map[string]*state.HelmState {
	repositories := map[string]state.RepositorySpec{}
	for k := range helmstate.Repositories {
		repo := helmstate.Repositories[k]
		if _, ok := repositories[repo.Name]; !ok {
			repositories[repo.Name] = repo
		}
	}

	addedRepos := map[string]map[string]bool{}

	helmStates := map[string]*state.HelmState{}
	for k := range helmstate.Releases {
		r := helmstate.Releases[k]
		ns := r.Namespace
		r.Namespace = ""
		if _, ok := helmStates[ns]; !ok {
			helmStates[ns] = &state.HelmState{
				ReleaseSetSpec: state.ReleaseSetSpec{
					OverrideNamespace: ns,
					Repositories:      []state.RepositorySpec{},
					Releases:          []state.ReleaseSpec{},
				},
			}
		}
		if _, ok := addedRepos[ns]; !ok {
			addedRepos[ns] = map[string]bool{}
		}

		hs := helmStates[ns]
		hs.Releases = append(hs.Releases, r)

		repoName := getRepoFromChart(r.Chart)
		if repoName == "." || repoName == ".." {
			// skip if repository is pointing at a local chart
			continue
		}
		if _, ok := addedRepos[ns][repoName]; !ok {
			hs.Repositories = append(hs.Repositories, repositories[repoName])
			addedRepos[ns][repoName] = true
		}
	}

	for ns := range helmStates {
		envSpecMap := map[string]state.EnvironmentSpec{}
		for key, env := range helmstate.Environments {
			var vals []interface{}
			vals = append(vals, env.Values...)

			envSpecMap[key] = state.EnvironmentSpec{
				Values: vals,
			}

		}
		helmStates[ns].Environments = envSpecMap
	}

	return helmStates
}

func getRepoFromChart(chartName string) string {
	return strings.Split(chartName, "/")[0]
}
