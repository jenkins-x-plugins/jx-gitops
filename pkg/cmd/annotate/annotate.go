package annotate

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/common"
	"github.com/jenkins-x/jx-gitops/pkg/kyamls"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	annotateLong = templates.LongDesc(`
		Annotates all kubernetes resources in the given directory tree
`)

	annotateExample = templates.Examples(`
		# updates recursively annotates all resources in the current directory 
		%s annotate myannotate=cheese another=thing
		# updates recursively all resources 
		%s annotate --dir myresource-dir foo=bar
	`)
)

// AnnotateOptions the options for the command
type Options struct {
	Dir      string
	Annotate string
}

// NewCmdUpdate creates a command object for the command
func NewCmdUpdateAnnotate() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "annotate",
		Short:   "Annotates all kubernetes resources in the given directory tree",
		Long:    annotateLong,
		Example: fmt.Sprintf(annotateExample, common.BinaryName, common.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := UpdateAnnotateInYamlFiles(o.Dir, args)
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "", ".", "the directory to recursively look for the *.yaml or *.yml files")

	return cmd, o
}

// UpdateAnnotateInYamlFiles updates the annotations in yaml files
func UpdateAnnotateInYamlFiles(dir string, annotations []string) error {
	modifyFn := func(node *yaml.RNode, path string) (bool, error) {
		sort.Strings(annotations)

		for _, a := range annotations {
			paths := strings.SplitN(a, "=", 2)
			k := paths[0]
			v := ""
			if len(paths) > 1 {
				v = paths[1]
			}

			err := node.PipeE(yaml.SetAnnotation(k, v))
			if err != nil {
				return false, errors.Wrapf(err, "failed to set annotation %s=%s", k, v)
			}
		}
		return true, nil
	}

	return kyamls.ModifyFiles(dir, modifyFn)
}
