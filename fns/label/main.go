package main

import (
	"fmt"
	"os"

	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
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

func main() {
	//o := &Options{}

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
	cmd.Example = fmt.Sprintf(labelExample, rootcmd.BinaryName, rootcmd.BinaryName)

	/*
		cmd := &cobra.Command{
			Use:     "label",
			Short:   ,
			Long:    labelLong,
			Example: fmt.Sprintf(labelExample, common.BinaryName, common.BinaryName),
			Run: func(cmd *cobra.Command, args []string) {
				err := UpdateLabelArgsInYamlFiles(o.Dir, args)
				helper.CheckErr(err)
			},
		}
		//cmd.Flags().StringVarP(&o.Dir, "dir", "", ".", "the directory to recursively look for the *.yaml or *.yml files")

	*/
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
