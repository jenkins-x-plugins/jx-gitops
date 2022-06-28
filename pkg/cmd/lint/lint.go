package lint

import (
	"fmt"
	"path/filepath"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/linter"
	"github.com/pkg/errors"
	"github.com/roboll/helmfile/pkg/state"
	"github.com/spf13/cobra"
)

var (
	splitLong = templates.LongDesc(`
		Lints the gitops files in the file system
`)

	splitExample = templates.Examples(`
		# lint files
		%s lint --dir .
	`)
)

// Options the options for the command
type Options struct {
	linter.Options

	Dir     string
	Verbose bool
	Linters []linter.Linter
}

// NewCmdLint creates a command object for the command
func NewCmdLint() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "lint",
		Short:   "Lints the gitops files in the file system",
		Long:    splitLong,
		Example: fmt.Sprintf(splitExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to recursively look for the *.yaml or *.yml files")
	return cmd, o
}

// Validate verifies the configuration
func (o *Options) Validate() error {
	o.Linters = append(o.Linters,
		linter.Linter{
			Path: filepath.Join(".jx", "gitops", v1alpha1.SourceConfigFileName),
			Linter: func(path string, test *linter.Test) error {
				return o.LintResource(path, test, &v1alpha1.SourceConfig{})
			},
		},
		linter.Linter{
			Path: filepath.Join("extensions", v1alpha1.PipelineCatalogFileName),
			Linter: func(path string, test *linter.Test) error {
				return o.LintResource(path, test, &v1alpha1.PipelineCatalog{})
			},
		},
		linter.Linter{
			Path: filepath.Join("extensions", v1alpha1.QuickstartsFileName),
			Linter: func(path string, test *linter.Test) error {
				return o.LintResource(path, test, &v1alpha1.Quickstarts{})
			},
		},
		linter.Linter{
			Path: v4beta1.RequirementsConfigFileName,
			Linter: func(path string, test *linter.Test) error {
				return o.LintResource(path, test, &v4beta1.Requirements{})
			},
		},
		linter.Linter{
			Path: "helmfile.yaml",
			Linter: func(path string, test *linter.Test) error {
				return o.LintYaml2Resource(path, test, &state.HelmState{})
			},
		},
	)
	return nil
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate")
	}

	return o.Lint(o.Linters, o.Dir)
}
