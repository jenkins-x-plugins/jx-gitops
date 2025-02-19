package validate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/helmfile/helmfile/pkg/state"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yaml2s"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Parses a helmfile and any nested helmfiles and validates they conform to a canonical directory structure for jx based around namespace
`)

	cmdExample = templates.Examples(`
		# Validates helmfile.yaml within the current directory
		%s helmfile validate
	`)
)

type Options struct {
	Dir       string
	Helmfile  string
	OutputDir string
}

func NewCmdHelmfileValidate() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "validate",
		Short:   "Validates helmfile.yaml against a jx canonical tree of helmfiles",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(_ *cobra.Command, _ []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&o.Helmfile, "helmfile", "", "", "the helmfile to template. Defaults to 'helmfile.yaml' in the directory")
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory that contains helmfile.yml")

	return cmd, o
}

// Validate validates the options and populates any missing values
func (o *Options) Validate() error {
	var err error
	if o.Helmfile == "" {
		o.Helmfile = filepath.Join(o.Dir, "helmfile.yaml")
	}
	if o.OutputDir == "" {
		o.OutputDir, err = os.MkdirTemp("", "")
		if err != nil {
			return errors.Wrapf(err, "failed to create temporary output directory")
		}
	}
	return nil
}

func (o Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate")
	}

	rootHelmState := state.HelmState{}

	err = yaml2s.LoadFile(o.Helmfile, &rootHelmState)
	if err != nil {
		return errors.Wrapf(err, "fail to load yaml file %s", o.Helmfile)
	}
	for _, nestedState := range rootHelmState.Helmfiles {
		err := o.validateSubHelmFile(nestedState.Path)
		if err != nil {
			return fmt.Errorf("failed to process nested helmfile %s with error %w", nestedState.Path, err)
		}
	}

	return nil
}

func (o Options) validateSubHelmFile(path string) error {
	targetNamespace, err := o.getSubHelmfileNamespace(path)
	if err != nil {
		return fmt.Errorf("failed to determine namespace from path %w", err)
	}

	helmState := state.HelmState{}
	err = yaml2s.LoadFile(filepath.Join(o.Dir, path), &helmState)
	if err != nil {
		return fmt.Errorf("failed to load helmfile - %w", err)
	}

	for k := range helmState.Releases {
		release := helmState.Releases[k]
		if release.Namespace != targetNamespace {
			return fmt.Errorf("namespace for release %s is %s does not match namespace of folder %s", release.Name, release.Namespace, targetNamespace)
		}
		chartRepo, err := getChartRepository(release.Chart)
		if err != nil {
			return fmt.Errorf("failed parsing repo name for chart %s", release.Chart)
		}
		if err := checkChartRepositoryExists(chartRepo, helmState.Repositories); err != nil {
			return fmt.Errorf("error finding chart repo for %s", chartRepo)
		}
	}
	return nil
}

func checkChartRepositoryExists(chartRepo string, repos []state.RepositorySpec) error {
	for k := range repos {
		repo := repos[k]
		if chartRepo == repo.Name {
			return nil
		}
	}
	return fmt.Errorf("repo for chart %s does not exist in repo list", chartRepo)
}

func getChartRepository(chart string) (string, error) {
	chartSplit := strings.Split(chart, "/")
	if len(chartSplit) != 2 {
		return "", fmt.Errorf("failed to determine chartname for %s", chart)
	}
	return chartSplit[0], nil
}

func (o Options) getSubHelmfileNamespace(path string) (string, error) {
	subHelmRel := path

	if filepath.IsAbs(path) {
		return "", fmt.Errorf("nested helmfiles should be specified relative to root helmfile")
	}

	subHelmRelDir, _ := filepath.Split(subHelmRel)
	subHelmRelDir = filepath.Clean(subHelmRelDir)

	directories := strings.Split(subHelmRelDir, string(filepath.Separator))

	if len(directories) != 2 {
		return "", fmt.Errorf("nested helmfile %s is not a grandchild of parent helmfile - should be stored within ./helmfiles/namespace", path)
	}

	return directories[1], nil
}
