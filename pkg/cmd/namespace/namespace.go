package namespace

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
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

		# In --dir-mode when a resource HAS DEFINED NAMESPACE already, but is in wrong directory
		# then it will be moved to a directory corresponding it's defined namespace
	`)
)

// Options the options for the command
type Options struct {
	kyamls.Filter
	Dir        string
	ClusterDir string
	Namespace  string
	DirMode    bool
}

// NewCmdUpdateNamespace creates a command object for the command
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
		_, err := UpdateNamespaceInYamlFiles(o.Dir, o.Dir, ns, &o.Filter, false)
		return err
	}

	return o.RunDirMode()
}

func (o *Options) RunDirMode() error {
	if o.Namespace != "" {
		return errors.Errorf("should not specify the --namespace option if you are running dir mode as the namespace is taken from the first child directory names")
	}
	flieList, err := ioutil.ReadDir(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to read dir %s", o.Dir)
	}

	var foundNamespacesInResources []string
	namespaces := []string{}
	for _, f := range flieList {
		if !f.IsDir() {
			continue
		}
		name := f.Name()

		dir := filepath.Join(o.Dir, name)
		foundNamespacesInResources, err = UpdateNamespaceInYamlFiles(o.Dir, dir, name, &o.Filter, true)
		if err != nil {
			return err
		}

		if stringhelpers.StringArrayIndex(namespaces, name) < 0 {
			namespaces = append(namespaces, name)
		}
		if len(foundNamespacesInResources) > 0 {
			namespaces = append(namespaces, foundNamespacesInResources...)
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
	// this namespace is created by `jx admin` and there is a possibly a difference in e.g. labels, so kubectl --prune can harm
	// temporary fix: https://kubernetes.slack.com/archives/C9MBGQJRH/p1647955769010979
	// before everybody will have https://github.com/jenkins-x/jx3-versions/pull/2963
	// todo: Find a better solution
	if ns == "jx-git-operator" {
		return nil
	}

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
func UpdateNamespaceInYamlFiles(rootDir, dir, ns string, filter *kyamls.Filter, shouldMoveFiles bool) ([]string, error) { //nolint:gocritic
	type docToMoveToOtherNs struct {
		path         string
		namespace    string
		oldNamespace string
	}
	var toMoveToNsDirectory []docToMoveToOtherNs
	var extraNamespacesFoundInResources []string

	modifyContentFn := func(node *yaml.RNode, path string) (bool, error) {
		kind := kyamls.GetKind(node, path)

		// ignore common cluster based resources
		if kyamls.IsClusterKind(kind) {
			return false, nil
		}

		// keep a namespace, and allow to move this file to a directory named with that namespace
		preserveOriginalNamespace := ShouldPreserveNamespace(node, path)
		if preserveOriginalNamespace {
			newNs := GetNamespaceToPreserveIfShouldKeepIt(node, path)
			if newNs != ns {
				toMoveToNsDirectory = append(toMoveToNsDirectory, docToMoveToOtherNs{path: path, namespace: newNs, oldNamespace: ns})
				extraNamespacesFoundInResources = append(extraNamespacesFoundInResources, newNs)
				return false, nil
			}
		}

		err := node.PipeE(yaml.LookupCreate(yaml.ScalarNode, "metadata", "namespace"), yaml.FieldSetter{StringValue: ns})
		if err != nil {
			return false, errors.Wrapf(err, "failed to set metadata.namespace to %s", ns)
		}
		return true, nil
	}

	err := kyamls.ModifyFiles(dir, modifyContentFn, *filter)
	if err != nil {
		return []string{}, errors.Wrapf(err, "failed to modify namespace to %s in dir %s", ns, dir)
	}

	if shouldMoveFiles {
		// files marked to keep their originally defined namespace will be moved to a directory
		// named same as .metadata.namespace
		for _, element := range toMoveToNsDirectory {
			if err := MoveToTargetNamespace(rootDir, element.path, element.namespace, element.oldNamespace, &osToolsImpl{}); err != nil {
				return []string{}, err
			}
		}
	}

	return extraNamespacesFoundInResources, nil
}

func MoveToTargetNamespace(rootDir, originalPath, namespace, oldNamespace string, osUtils osTools) error {
	// normalize to absolute paths
	rootDir, _ = filepath.Abs(rootDir)
	originalPath, _ = filepath.Abs(originalPath)
	rootDir = strings.TrimSuffix(rootDir, "/") // normalize

	// extract subdirectory structure in existing namespace
	relativePath := originalPath[len(rootDir+"/"+oldNamespace):]

	newNamespacedDirPath := rootDir + "/" + namespace
	newNamespacedFilePath := newNamespacedDirPath + relativePath

	if err := osUtils.MkdirAll(filepath.Dir(newNamespacedFilePath), 0755); err != nil {
		return errors.Wrapf(err, "cannot create a directory for target namespace '%s'", namespace)
	}

	log.Logger().Infof("Moving '%s' to '%s' as it had defined .metadata.namespace", originalPath, newNamespacedFilePath)
	if err := osUtils.Rename(originalPath, newNamespacedFilePath); err != nil {
		return errors.Wrap(err, "cannot move YAML file to target namespace directory")
	}

	return nil
}

func ShouldPreserveNamespace(node *yaml.RNode, path string) bool {
	return GetNamespaceToPreserveIfShouldKeepIt(node, path) != ""
}

func GetNamespaceToPreserveIfShouldKeepIt(node *yaml.RNode, path string) string {
	existingNs := kyamls.GetNamespace(node, path)
	if existingNs != "" {
		return existingNs
	}
	return ""
}

type osTools interface {
	MkdirAll(path string, perm os.FileMode) error
	Rename(oldpath, newpath string) error
}
type osToolsImpl struct{}

func (o *osToolsImpl) MkdirAll(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (o *osToolsImpl) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}
