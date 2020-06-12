package namespace

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/common"
	"github.com/jenkins-x/jx-gitops/pkg/mapslices"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
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
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		// lets load the file
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to load file %s", path)
		}

		ms := yaml.MapSlice{}
		err = yaml.Unmarshal(data, &ms)
		if err != nil {
			return errors.Wrapf(err, "failed to unmarshal YAML file %s", path)
		}

		kind, _, err := mapslices.NestedString(ms, "kind")
		if err != nil {
			return errors.Wrapf(err, "failed to find kind in path %s", path)
		}
		// ignore common cluster based resources
		if kind == "" || kind == "Namespace" || strings.HasPrefix(kind, "Cluster") {
			return nil
		}

		fields := []string{"metadata", "namespace"}
		ms, flag, err := mapslices.SetNestedField(ms, ns, fields...)
		if err != nil {
			return errors.Wrapf(err, "failed to set fields %#v to value %v", fields, ns)
		}
		if !flag {
			return nil
		}
		data, err = yaml.Marshal(ms)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal modified file %s", path)
		}

		err = ioutil.WriteFile(path, data, util.DefaultFileWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to save %s", path)
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to set namespace to %s in dir %s", ns, dir)
	}
	return nil
}
