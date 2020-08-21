package move

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/kyamls"
	"github.com/jenkins-x/jx-helpers/pkg/options"
	"github.com/jenkins-x/jx-helpers/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/pkg/yaml2s"
	"github.com/jenkins-x/jx-helpers/pkg/yamls"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
	"github.com/roboll/helmfile/pkg/state"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	pathSeparator = string(os.PathSeparator)
)

var (
	namespaceLong = templates.LongDesc(`
		Moves the generated template files from 'helmfile template' into the right gitops directory

The output of 'helmfile template' ignores the namespace specified in the 'helmfile.yaml' and there is a dummy top level directory.

So this command applies the namespace to all the generated resources and then moves the namespaced resources into the config-root/namespaces/$ns/$releaseName directory
and then moves any CRDs or cluster level resources into 'config-root/cluster/$releaseName'
`)

	namespaceExample = templates.Examples(`
		# moves the generated files in 'tmp' to the config root dir
		%s helmfile move --dir config-root --from tmp
	`)
)

// NamespaceOptions the options for the command
type Options struct {
	kyamls.Filter
	Helmfile             string
	Dir                  string
	OutputDir            string
	ClusterDir           string
	ClusterNamespacesDir string
	NamespacesDir        string
	SingleNamespace      string
	HelmState            *state.HelmState
}

// NewCmdHelmfileMove creates a command object for the command
func NewCmdHelmfileMove() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "move",
		Aliases: []string{"mv"},
		Short:   "Moves the generated template files from 'helmfile template' into the right gitops directory",
		Long:    namespaceLong,
		Example: fmt.Sprintf(namespaceExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Helmfile, "helmfile", "f", "helmfile.yaml", "the 'helmfile.yaml' file to find the namespaces for each release name")
	cmd.Flags().StringVarP(&o.Dir, "dir", "", "", "the directory containing the generated resources")
	cmd.Flags().StringVarP(&o.OutputDir, "output-dir", "o", "config-root", "the output directory")
	o.Filter.AddFlags(cmd)
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	if o.HelmState == nil {
		o.HelmState = &state.HelmState{}
		if o.Helmfile == "" {
			return options.MissingOption("helmfile")
		}
		err := yaml2s.LoadFile(o.Helmfile, o.HelmState)
		if err != nil {
			return errors.Wrapf(err, "failed to load helmfile %s", o.Helmfile)
		}
	}
	if o.ClusterDir == "" {
		o.ClusterDir = filepath.Join(o.OutputDir, "cluster")
	}
	if o.NamespacesDir == "" {
		o.NamespacesDir = filepath.Join(o.OutputDir, "namespaces")
	}
	if o.ClusterNamespacesDir == "" {
		o.ClusterNamespacesDir = filepath.Join(o.ClusterDir, "namespaces")
		err := os.MkdirAll(o.ClusterNamespacesDir, files.DefaultDirWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to create cluster namespaces dir %s", o.ClusterNamespacesDir)
		}
	}

	g := filepath.Join(o.Dir, "*/*")
	fileNames, err := filepath.Glob(g)
	if err != nil {
		return errors.Wrapf(err, "failed to glob files %s", g)
	}

	var namespaces []string
	for _, dir := range fileNames {
		log.Logger().Infof("processing chart dir %s", dir)

		exists, err := files.DirExists(dir)
		if err != nil {
			return errors.Wrapf(err, "failed to check if path exists %s", dir)
		}
		if !exists {
			continue
		}

		_, name := filepath.Split(dir)
		ns := o.findNamespaceForReleaseName(name)
		if ns == "" {
			return errors.Errorf("could not find namespace for release name %s in path %s", name, dir)
		}
		if stringhelpers.StringArrayIndex(namespaces, ns) < 0 {
			namespaces = append(namespaces, ns)
		}

		err = o.moveFilesToClusterOrNamespacesFolder(dir, ns, name)
		if err != nil {
			return errors.Wrapf(err, "failed to ")
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

func (o *Options) findNamespaceForReleaseName(name string) string {
	if o.SingleNamespace != "" {
		return o.SingleNamespace
	}
	ns := ""
	for _, r := range o.HelmState.Releases {
		if r.Name == name {
			ns = r.Namespace
			break
		}
	}
	return ns
}

func (o *Options) lazyCreateNamespaceResource(ns string) error {
	dir := filepath.Dir(o.ClusterNamespacesDir)

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

	fileName := filepath.Join(o.ClusterNamespacesDir, ns+".yaml")

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

	log.Logger().Infof("no Namespace resource %s so created file %s", termcolor.ColorInfo(ns), termcolor.ColorInfo(fileName))
	return nil
}

func (o *Options) moveFilesToClusterOrNamespacesFolder(dir string, ns string, releaseName string) error {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return errors.Wrapf(err, "failed to calculate relative path of %s from %s", path, dir)
		}

		// lets remove the last but one dir if its 'templates'
		paths := strings.Split(rel, pathSeparator)
		i := len(paths) - 2
		if i >= 0 {
			if paths[i] == "templates" || paths[i] == "crds" {
				paths = append(paths[0:i], paths[i+1])
				rel = strings.Join(paths, pathSeparator)
			}
		}

		node, err := yaml.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to load file %s", path)
		}

		kind := kyamls.GetKind(node, path)
		outDir := filepath.Join(o.ClusterDir, ns, releaseName)
		if !kyamls.IsClusterKind(kind) {
			err := node.PipeE(yaml.LookupCreate(yaml.ScalarNode, "metadata", "namespace"), yaml.FieldSetter{StringValue: ns})
			if err != nil {
				return errors.Wrapf(err, "failed to set metadata.namespace to %s for path %s", ns, path)
			}
			outDir = filepath.Join(o.NamespacesDir, ns, releaseName)
		}

		outFile := filepath.Join(outDir, rel)
		parentDir := filepath.Dir(outFile)
		err = os.MkdirAll(parentDir, files.DefaultDirWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to create dir %s", parentDir)
		}

		err = yaml.WriteFile(node, outFile)
		if err != nil {
			return errors.Wrapf(err, "failed to save %s", outFile)
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to modify namespace to %s for release %s in dir %s", ns, releaseName, dir)
	}
	return nil
}
