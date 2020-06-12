package label

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/common"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	labelLong = templates.LongDesc(`
		Updates all kubernetes resources in the given directory tree to add/override the given label
`)

	labelExample = templates.Examples(`
		# updates recursively labels all resources in the current directory 
		%s step update label mylabel=cheese another=thing
		# updates recursively all resources 
		%s step update label --dir myresource-dir foo=bar
	`)
)

// LabelOptions the options for the command
type Options struct {
	Dir   string
	Label string
}

// NewCmdUpdate creates a command object for the command
func NewCmdUpdateLabel() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "label",
		Short:   "Updates all kubernetes resources in the given directory tree to add/override the given label",
		Long:    labelLong,
		Example: fmt.Sprintf(labelExample, common.BinaryName, common.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := UpdateLabelInYamlFiles(o.Dir, args)
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "", ".", "the directory to recursively look for the *.yaml or *.yml files")

	return cmd, o
}

// UpdateLabelInYamlFiles updates the labels in yaml files
func UpdateLabelInYamlFiles(dir string, labels []string) error {
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

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}
		node, err := yaml.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to load file %s", path)
		}

		modified, err := modifyFn(node, path)
		if err != nil {
			return errors.Wrapf(err, "failed to modify file %s", path)
		}

		if !modified {
			return nil
		}

		err = yaml.WriteFile(node, path)
		if err != nil {
			return errors.Wrapf(err, "failed to save %s", path)
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to set labels %#v in dir %s", labels, dir)
	}
	return nil
}

/*
func dummy() (*cobra.Command, *Options) {
	o := &Options{}

	resourceList := &framework.ResourceList{}
	cmd := framework.Command(resourceList, func() error {
		fmt.Println("TODO: starting up....")
		// cmd.Execute() will parse the ResourceList.functionConfig into cmd.Flags from
		// the ResourceList.functionConfig.data field.

		args := resourceList.Flags.Args()
		log.Logger().Infof("invoked with args %#v", args)

		for i := range resourceList.Items {
			// modify the resources using the kyaml/yaml library:
			// https://pkg.go.dev/sigs.k8s.io/kustomize/kyaml/yaml
			filter := yaml.SetLabel("value", "dummy")
			if err := resourceList.Items[i].PipeE(filter); err != nil {
				return err
			}
		}
		return nil
	})

	cmd.Use = "label"
	cmd.Short = "Updates all kubernetes resources in the given directory tree to add/override the given label"
	cmd.Long = labelLong
	cmd.Example = fmt.Sprintf(labelExample, common.BinaryName, common.BinaryName)

	return &cmd, o
}
*/
