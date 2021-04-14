package yset

import (
	"fmt"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	annotateLong = templates.LongDesc(`
		Modifies one or more yaml files using a path expression
`)

	annotateExample = templates.Examples(`
		# sets the foo.bar=abc in the file foo.yaml
		jx gitops yset --file foo.yaml --path foo.bar --value abc
	`)
)

// AnnotateOptions the options for the command
type Options struct {
	Files []string
	Path  string
	Value string
}

// NewCmdUpdate creates a command object for the command
func NewCmdYSet() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "yset",
		Short:   "Modifies a value in a YAML file at a given path",
		Long:    annotateLong,
		Example: fmt.Sprintf(annotateExample, rootcmd.BinaryName, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Path, "path", "p", "", "the path expression to modify (separated by dots)")
	cmd.Flags().StringVarP(&o.Value, "value", "v", "", "the value to modify")
	cmd.Flags().StringArrayVarP(&o.Files, "file", "f", nil, "the file(s) to process")
	return cmd, o
}

func (o *Options) Run() error {
	if len(o.Files) == 0 {
		return options.MissingOption("file")
	}
	if o.Path == "" {
		return options.MissingOption("path")
	}
	if o.Value == "" {
		return options.MissingOption("value")
	}

	for _, fileName := range o.Files {
		node, err := yaml.ReadFile(fileName)
		if err != nil {
			return errors.Wrapf(err, "failed to load file %s", fileName)
		}

		v := o.Value
		vn := yaml.NewScalarRNode(v)

		paths := strings.Split(o.Path, ".")
		lastIdx := len(paths) - 1
		k := paths[lastIdx]
		if lastIdx > 0 {
			paths = paths[0:lastIdx]
			_, err = node.Pipe(
				yaml.PathGetter{Path: paths, Create: yaml.MappingNode},
				yaml.FieldSetter{Name: k, Value: vn})
			if err != nil {
				return errors.Wrapf(err, "failed to modify node %s set path %s=%s", strings.Join(paths, "."), k, v)
			}
		} else {
			_, err = node.Pipe(yaml.FieldSetter{Name: k, Value: vn})
			if err != nil {
				return errors.Wrapf(err, "failed to set path %s=%s", k, v)
			}
		}

		err = yaml.WriteFile(node, fileName)
		if err != nil {
			return errors.Wrapf(err, "failed to save %s", fileName)
		}
	}
	return nil
}
