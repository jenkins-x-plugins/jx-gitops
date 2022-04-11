package namespace

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	namespaceLong = templates.LongDesc(`
		Updates all kubernetes resources in the given directory to the given namespace
`)

	namespaceExample = templates.Examples(`
		# updates the namespace of all the yaml resources in the given directory
		%s namespace -n cheese --dir .


		# sets the namespace property to the name of the child directory inside of 'config-root/namespaces'
		# e.g. so that the files 'config-root/namespaces/cheese/*.yaml' get set to namespace 'cheese' 
		# and 'config-root/namespaces/wine/*.yaml' are set to 'wine'
		%s namespace --dir-mode --dir config-root/namespaces
	`)
)

// NamespaceOptions the options for the command
type Options struct {
	kyamls.Filter
	Dir        string
	ClusterDir string
	Namespace  string
	DirMode    bool
}

// NewCmdUpdate creates a command object for the command
func NewCmdUpdateNamespace() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "namespace",
		Aliases: []string{"ns"},
		Short:   "Updates all kubernetes resources in the given directory to the given namespace",
		Long:    namespaceLong,
		Example: fmt.Sprintf(namespaceExample, rootcmd.BinaryName, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "", ".", "the directory to recursively look for the namespaced *.yaml or *.yml files to set the namespace on")
	cmd.Flags().StringVarP(&o.ClusterDir, "cluster-dir", "", "", "the directory to recursively look for the *.yaml or *.yml files")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", "the namespace to modify the resources to")
	cmd.Flags().BoolVarP(&o.DirMode, "dir-mode", "", false, "assumes the first child directory is the name of the namespace to use")
	o.Filter.AddFlags(cmd)
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	ns := o.Namespace
	if o.ClusterDir == "" {
		// lets navigate relative to the namespaces dir
		o.ClusterDir = filepath.Join(o.Dir, "..", "cluster", "namespaces")
		err := os.MkdirAll(o.ClusterDir, files.DefaultDirWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to create cluster namespaces dir %s", o.ClusterDir)
		}
	}
	if !o.DirMode {
		if ns == "" {
			return options.MissingOption("namespace")
		}
		return UpdateNamespaceInYamlFiles(o.Dir, ns, o.Filter)
	}

	return o.RunDirMode()
}

func (o *Options) RunDirMode() error {
	if o.Namespace != "" {
		return errors.Errorf("should not specify the --namespace option if you are running dir mode as the namespace is taken from the first child directory names")
	}
	flieList, err := os.ReadDir(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to read dir %s", o.Dir)
	}

	namespaces := []string{}
	for _, f := range flieList {
		if !f.IsDir() {
			continue
		}
		name := f.Name()

		dir := filepath.Join(o.Dir, name)
		err = UpdateNamespaceInYamlFiles(dir, name, o.Filter)
		if err != nil {
			return err
		}

		if stringhelpers.StringArrayIndex(namespaces, name) < 0 {
			namespaces = append(namespaces, name)
		}
	}

	// now lets lazy create any namespace resources which don't exist in the cluster dir
	for _, ns := range namespaces {
		err = o.lazyCreateNamespaceResource(ns)
		if err != nil {
			return errors.Wrapf(err, "failed to lazily create namespace resource %s", ns)
		}
	}
	return nil
}

func (o *Options) lazyCreateNamespaceResource(ns string) error {
	dir := filepath.Dir(o.ClusterDir)

	found := false

	modifyFn := func(node *yaml.RNode, path string) (bool, error) {
		kind := kyamls.GetKind(node, path)
		if kind == "Namespace" {
			name := kyamls.GetName(node, path)
			if name == ns {
				found = true
			}
		}
		return false, nil
	}

	filter := kyamls.Filter{
		Kinds: []string{"Namespace"},
	}
	err := kyamls.ModifyFiles(dir, modifyFn, filter)
	if err != nil {
		return errors.Wrapf(err, "failed to walk namespaces in dir %s", dir)
	}
	if found {
		return nil
	}

	fileName := filepath.Join(o.ClusterDir, ns+".yaml")

	namespace := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: ns,
			Labels: map[string]string{
				"name": ns,
			},
		},
	}
	err = yamls.SaveFile(namespace, fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", fileName)
	}

	log.Logger().Debugf("no Namespace resource %s so created file %s", termcolor.ColorInfo(ns), termcolor.ColorInfo(fileName))
	return nil
}

// UpdateNamespaceInYamlFiles updates the namespace in yaml files
func UpdateNamespaceInYamlFiles(dir string, ns string, filter kyamls.Filter) error { //nolint:gocritic
	modifyFn := func(node *yaml.RNode, path string) (bool, error) {
		kind := kyamls.GetKind(node, path)

		// ignore common cluster based resources
		if kyamls.IsClusterKind(kind) {
			return false, nil
		}

		err := node.PipeE(yaml.LookupCreate(yaml.ScalarNode, "metadata", "namespace"), yaml.FieldSetter{StringValue: ns})
		if err != nil {
			return false, errors.Wrapf(err, "failed to set metadata.namespace to %s", ns)
		}
		return true, nil
	}

	err := kyamls.ModifyFiles(dir, modifyFn, filter)
	if err != nil {
		return errors.Wrapf(err, "failed to modify namespace to %s in dir %s", ns, dir)
	}
	return nil
}
