package get

import (
	"os"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/plugins"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/table"
	"github.com/spf13/cobra"
)

// Options is the start of the data required to perform the operation.
// As new fields are added, add them here instead of
// referencing the cmd.Flags()
type Options struct {
}

var (
	cmdLong = templates.LongDesc(`
		Display the binary plugins

`)

	cmdExample = templates.Examples(`
		# list all binary plugins
		jx plugin get
	`)
)

// NewCmdPluginGet creates the command
func NewCmdPluginGet() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "get",
		Short:   "Display the binary plugins",
		Long:    cmdLong,
		Example: cmdExample,
		Aliases: []string{"list", "ls"},
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	return cmd, o
}

// Run implements this command
func (o *Options) Run() error {
	out := os.Stdout
	t := table.CreateTable(out)
	t.AddRow("NAME", "VERSION")

	for i := range plugins.Plugins {
		p := &plugins.Plugins[i]
		t.AddRow(p.Name, p.Spec.Version)
	}
	t.Render()
	return nil
}
