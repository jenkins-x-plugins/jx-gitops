package move

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/helmfile/helmfile/pkg/state"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/helmhelpers"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (
	// HelmReleaseNameAnnotation the annotation added by helm to denote a release name
	HelmReleaseNameAnnotation      = "meta.helm.sh/release-name"
	HelmReleaseNameSpaceAnnotation = "meta.helm.sh/release-namespace"

	pathSeparator = string(os.PathSeparator)
)

var (
	namespaceLong = templates.LongDesc(`
		Moves the generated template files from 'helmfile template' into the right gitops directory.

		The output of 'helmfile template' ignores the namespace specified in the 'helmfile.yaml' and there is a dummy top level directory.

		So by default this command applies the namespace to all the generated resources

		Then it moves the namespaced resources into the config-root/namespaces/$ns/$releaseName directory
		and any CRDs or cluster level resources into 'config-root/cluster/$releaseName'.

		If supplied with --dir-includes-release-name then by default we will annotate the resources with the annotations "app.kubernetes.io/instance" to preserve the helm release name.

		The annotation "meta.helm.sh/release-namespace" will be added by default and contain the namespace specified in the release.
`)

	namespaceExample = templates.Examples(`
		# moves the generated files in 'tmp' to the config root dir
		%s helmfile move --dir config-root --from tmp
	`)
)

// NamespaceOptions the options for the command
type Options struct {
	kyamls.Filter
	Dir                          string
	OutputDir                    string
	ClusterDir                   string
	ClusterNamespacesDir         string
	ClusterResourcesDir          string
	CustomResourceDefinitionsDir string
	NamespacesDir                string
	DirIncludesReleaseName       bool
	OverrideNamespace            bool
	AnnotateReleaseNames         bool
	AnnotateReleaseNameSpace     bool
	HelmState                    *state.HelmState
	NamespacedKind               map[string]bool
	ResourcesToMove              []ResourceToMove
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
		Run: func(_ *cobra.Command, _ []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "", "", "the directory containing the generated resources")
	cmd.Flags().StringVarP(&o.OutputDir, "output-dir", "o", "config-root", "the output directory")
	cmd.Flags().BoolVarP(&o.DirIncludesReleaseName, "dir-includes-release-name", "", false, "the directory containing the generated resources has a path segment that is the release name")
	cmd.Flags().BoolVarP(&o.AnnotateReleaseNames, "annotate-release-name", "", true, "if using --dir-includes-release-name layout then lets add the 'meta.helm.sh/release-name' annotation to record the helm release name")
	cmd.Flags().BoolVarP(&o.AnnotateReleaseNameSpace, "annotate-release-namespace", "", true, "add the 'meta.helm.sh/release-namespace' annotation to record the helm release namespace")
	cmd.Flags().BoolVarP(&o.OverrideNamespace, "override-namespace", "", true, "applies the namespace specified in helmfile to all the generated resources")

	o.Filter.AddFlags(cmd)
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	if o.ClusterDir == "" {
		o.ClusterDir = filepath.Join(o.OutputDir, "cluster")
	}
	if o.NamespacesDir == "" {
		o.NamespacesDir = filepath.Join(o.OutputDir, "namespaces")
	}
	if o.ClusterResourcesDir == "" {
		o.ClusterResourcesDir = filepath.Join(o.ClusterDir, "resources")
		err := os.MkdirAll(o.ClusterResourcesDir, files.DefaultDirWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to create cluster resources dir %s", o.ClusterResourcesDir)
		}
	}
	if o.ClusterNamespacesDir == "" {
		o.ClusterNamespacesDir = filepath.Join(o.ClusterDir, "namespaces")
		err := os.MkdirAll(o.ClusterNamespacesDir, files.DefaultDirWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to create cluster namespaces dir %s", o.ClusterNamespacesDir)
		}
	}
	if o.CustomResourceDefinitionsDir == "" {
		o.CustomResourceDefinitionsDir = filepath.Join(o.OutputDir, "customresourcedefinitions")
	}

	globPattern := "*/*"
	if o.DirIncludesReleaseName {
		globPattern = "*/*/*"
	}
	g := filepath.Join(o.Dir, globPattern)
	fileNames, err := filepath.Glob(g)
	if err != nil {
		return errors.Wrapf(err, "failed to glob files %s", g)
	}

	var namespaces []string
	o.NamespacedKind = make(map[string]bool)
	if !kube.IsNoKubernetes() {
		client, err := kube.LazyCreateKubeClient(nil)
		if err != nil {
			log.Logger().Errorf("Failed to create k8s client: %v", err)
			goto noKube
		}
		apiResourceLists, err := client.Discovery().ServerPreferredResources()
		if err != nil {
			log.Logger().Errorf("Failed to fetch api resources: %v", err)
		}
		for i := range apiResourceLists {
			resources := apiResourceLists[i].APIResources
			for j := range resources {
				o.NamespacedKind[resources[j].Kind] = resources[j].Namespaced
			}
		}
	}
noKube:
	o.ResourcesToMove = make([]ResourceToMove, 0, len(fileNames))
	for _, dir := range fileNames {
		log.Logger().Debugf("processing chart dir %s", dir)

		exists, err := files.DirExists(dir)
		if err != nil {
			return errors.Wrapf(err, "failed to check if path exists %s", dir)
		}
		if !exists {
			continue
		}

		var ns, releaseName, chartName string

		relDir, _ := filepath.Rel(o.Dir, dir)

		parts := strings.Split(relDir, string(os.PathSeparator))

		if o.DirIncludesReleaseName {
			// {{.Release.Namespace}}/{{.Release.Name}}/chartName
			ns, releaseName, chartName = parts[0], parts[1], parts[2]
		} else {
			// {{.Release.Namespace}}/chartName
			ns, releaseName, chartName = parts[0], parts[1], parts[1]
		}
		namespaces = append(namespaces, ns)

		err = o.moveFilesToClusterOrNamespacesFolder(dir, ns, releaseName, chartName)
		if err != nil {
			return errors.Wrapf(err, "failed to ")
		}
	}
	for _, res := range o.ResourcesToMove {
		isNamespaced, ok := o.NamespacedKind[res.kind]
		if !ok {
			isNamespaced = !kyamls.IsClusterKind(res.kind)
			o.NamespacedKind[res.kind] = isNamespaced
			if !kube.IsNoKubernetes() {
				not := ""
				if !isNamespaced {
					not = " not"
				}
				log.Logger().Errorf("the server doesn't have resource of kind %s. Assuming it is%s namespaced.", res.kind, not)
			}
		}
		outDir := filepath.Join(o.ClusterResourcesDir, res.namespace, res.pathname)

		if isNamespaced {
			setNS := true
			namespaceLookup := yaml.LookupCreate(yaml.ScalarNode, "metadata", "namespace")
			if !o.OverrideNamespace {
				nsNode, _ := res.node.Pipe(namespaceLookup)
				nsNodeText, _ := nsNode.String()
				setNS = nsNodeText == ""
			}
			if setNS {
				err = res.node.PipeE(namespaceLookup, yaml.FieldSetter{StringValue: res.namespace})

				if err != nil {
					return errors.Wrapf(err, "failed to set metadata.namespace to %s for path %s", res.namespace, res.path)
				}
			}

			outDir = filepath.Join(o.NamespacesDir, res.namespace, res.pathname)
		} else {
			err := res.node.PipeE(yaml.Lookup("metadata"), yaml.FieldClearer{Name: "namespace"})
			if err != nil {
				return errors.Wrapf(err, "failed to remove metadata.namespace for path %s", res.path)
			}
		}
		err = o.writeNodeToDir(outDir, res.rel, res.node)
		if err != nil {
			return err
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

	log.Logger().Debugf("no Namespace resource %s so created file %s", termcolor.ColorInfo(ns), termcolor.ColorInfo(fileName))
	return nil
}

func (o *Options) moveFilesToClusterOrNamespacesFolder(dir, ns, releaseName, chartName string) error {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error { //nolint:staticcheck
		if info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}
		rel, err := filepath.Rel(dir, path) //nolint:staticcheck
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

		// lets check for empty yaml files
		data, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to read file %s", path)
		}
		if helmhelpers.IsWhitespaceOrComments(string(data)) {
			log.Logger().Infof("ignoring empty yaml file %s", path)
			return nil
		}

		node, err := yaml.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to load YAML file %s", path)
		}

		// pathName is always prefixed with chartName but lets also remove any duplication
		var pathName string
		if chartName == releaseName {
			pathName = chartName
		} else if strings.HasPrefix(releaseName, chartName) {
			pathName = releaseName
		} else {
			pathName = fmt.Sprintf("%s-%s", chartName, releaseName)
		}

		setAnnotations := make(map[string]string)
		if o.AnnotateReleaseNames {
			setAnnotations[HelmReleaseNameAnnotation] = releaseName
		}
		if o.AnnotateReleaseNameSpace {
			setAnnotations[HelmReleaseNameSpaceAnnotation] = ns
		}
		for k, newValue := range setAnnotations {
			v, err := node.Pipe(yaml.GetAnnotation(k))
			if err != nil {
				return errors.Wrapf(err, "failed to get annotation %s for path %s", k, path)
			}
			if v == nil {
				err = node.PipeE(yaml.SetAnnotation(k, newValue))
				if err != nil {
					return errors.Wrapf(err, "failed to set annotation %s to %s for path %s", k, newValue, path)
				}
			}
		}

		kind := kyamls.GetKind(node, path)
		if kind == "" {
			return fmt.Errorf("No kind in %s", path)
		}

		if kyamls.IsCustomResourceDefinition(kind) {
			namespaced := kyamls.GetStringField(node, path, "spec", "scope") == "Namespaced"
			name := kyamls.GetStringField(node, path, "spec", "names", "kind")
			log.Logger().Debugf("CRD %s: namespaced = %v", name, namespaced)
			o.NamespacedKind[name] = namespaced
			return o.writeNodeToDir(filepath.Join(o.CustomResourceDefinitionsDir, ns, pathName), rel, node)
		}
		o.ResourcesToMove = append(o.ResourcesToMove, ResourceToMove{
			kind:      kind,
			node:      node,
			path:      path,
			pathname:  pathName,
			rel:       rel,
			namespace: ns,
		})
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to modify namespace to %s for release %s in dir %s", ns, releaseName, dir)
	}
	return nil
}

type ResourceToMove struct {
	kind      string
	path      string
	rel       string
	node      *yaml.RNode
	pathname  string
	namespace string
}

func (o *Options) writeNodeToDir(outDir, rel string, node *yaml.RNode) error {
	outFile := filepath.Join(outDir, rel)
	parentDir := filepath.Dir(outFile)
	err := os.MkdirAll(parentDir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create dir %s", parentDir)
	}

	err = yaml.WriteFile(node, outFile)
	if err != nil {
		return errors.Wrapf(err, "failed to save %s", outFile)
	}
	return nil
}
