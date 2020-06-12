package namespace

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/common"
	"github.com/jenkins-x/jx-gitops/pkg/kyamls"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	namespaceLong = templates.LongDesc(`
		Updates all kubernetes resources in the given directory to the given namespace
`)

	namespaceExample = templates.Examples(`
		# updates the namespace of all the yaml resources in the given directory
		%s step update namespace -n cheese --dir .
	`)
)

// NamespaceOptions the options for the command
type Options struct {
	Dir       string
	Namespace string
}

// NewCmdUpdate creates a command object for the command
func NewCmdUpdateNamespace() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "namespace",
		Aliases: []string{"ns"},
		Short:   "Updates all kubernetes resources in the given directory to the given namespace",
		Long:    namespaceLong,
		Example: fmt.Sprintf(namespaceExample, common.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "", ".", "the directory to recursively look for the *.yaml or *.yml files")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", "the namespace to modify the resources to")

	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	if o.Namespace == "" {
		return util.MissingOption("namespace")
	}
	return UpdateNamespaceInYamlFiles(o.Dir, o.Namespace)
}

// UpdateNamespaceInYamlFiles updates the namespace in yaml files
func UpdateNamespaceInYamlFiles(dir string, ns string) error {
	modifyFn := func(node *yaml.RNode, path string) (bool, error) {
		kind := ""
		kindNode := node.Field("kind")
		if kindNode != nil && kindNode.Value != nil {
			var err error
			kind, err = kindNode.Value.String()
			if err != nil {
				return false, errors.Wrapf(err, "failed to find kind")
			}
		}

		// ignore common cluster based resources
		if kind == "" || kind == "Namespace" || strings.HasPrefix(kind, "Cluster") {
			return false, nil
		}

		err := node.PipeE(yaml.Lookup("metadata", "namespace"), yaml.FieldSetter{StringValue: ns})
		if err != nil {
			return false, errors.Wrapf(err, "failed to set metadata.namespace to %s", ns)
		}
		return true, nil
	}

	return kyamls.ModifyFiles(dir, modifyFn)
}
