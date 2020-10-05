package label

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	cmdLong = templates.LongDesc(`
		Updates all kubernetes resources in the given directory tree to add/override the given label
`)

	cmdExample = templates.Examples(`
		# updates recursively labels all resources in the current directory 
		%s label mylabel=cheese another=thing
		# updates recursively all resources 
		%s label --dir myresource-dir foo=bar
	`)
)

// Options the options for the command
type Options struct {
	kyamls.Filter
	Dir   string
	Label string
}

// NewCmdUpdate creates a command object for the command
func NewCmdUpdateLabel() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "label",
		Short:   "Updates all kubernetes resources in the given directory tree to add/override the given label",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := UpdateLabelInYamlFiles(o.Dir, args, o.Filter)
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "", ".", "the directory to recursively look for the *.yaml or *.yml files")
	o.Filter.AddFlags(cmd)
	return cmd, o
}

// UpdateLabelInYamlFiles updates the labels in yaml files
func UpdateLabelInYamlFiles(dir string, labels []string, filter kyamls.Filter) error {
	modifyFn := func(node *yaml.RNode, path string) (bool, error) {
		sort.Strings(labels)

		for _, a := range labels {
			paths := strings.SplitN(a, "=", 2)
			k := paths[0]
			v := ""
			if len(paths) > 1 {
				v = paths[1]
			}

			err := node.PipeE(yaml.SetLabel(k, v))
			if err != nil {
				return false, errors.Wrapf(err, "failed to set label %s=%s", k, v)
			}
		}
		return true, nil
	}

	return kyamls.ModifyFiles(dir, modifyFn, filter)
}
