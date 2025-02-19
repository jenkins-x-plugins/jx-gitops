package patch

import (
	"context"
	"fmt"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-kube-client/v3/pkg/kubeclient"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
)

var (
	info = termcolor.ColorInfo

	annotateLong = templates.LongDesc(`
		Annotates all kubernetes resources in the given directory tree
`)

	annotateExample = templates.Examples(`
		# updates recursively annotates all resources in the current directory 
		%s annotate myannotate=cheese another=thing
		# updates recursively all resources 
		%[1]s annotate --dir myresource-dir foo=bar
	`)
)

// AnnotateOptions the options for the command
type Options struct {
	Namespace     string
	Group         string
	Version       string
	Kind          string
	Selector      string
	Name          string
	Type          string
	Data          string
	DynamicClient dynamic.Interface
}

// NewCmdPatch creates a command object for the command
func NewCmdPatch() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "patch",
		Short:   "Patches the given resources",
		Long:    annotateLong,
		Example: fmt.Sprintf(annotateExample, rootcmd.BinaryName),
		Run: func(_ *cobra.Command, _ []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Namespace, "ns", "n", "", "the namespace to find the resources to copy. Defaults to the current namespace")
	cmd.Flags().StringVarP(&o.Group, "group", "g", "apps", "the API group such as 'apps' for Deployemnts")
	cmd.Flags().StringVarP(&o.Version, "version", "", "v1", "the API version of the resources to copy")
	cmd.Flags().StringVarP(&o.Kind, "kind", "k", "deployments", "the kind name")
	cmd.Flags().StringVarP(&o.Selector, "selector", "l", "", "the label selector to find the resources to copy")
	cmd.Flags().StringVarP(&o.Name, "name", "", "", "the name of the resource to copy instead of a selector")
	cmd.Flags().StringVarP(&o.Type, "type", "", "", "the patch type such as 'yaml' or 'json'")
	cmd.Flags().StringVarP(&o.Data, "data", "d", "", "the patch data to apply as json or yaml")
	return cmd, o
}

func (o *Options) Run() error {
	if o.Selector == "" && o.Name == "" {
		return options.MissingOption("selector")
	}
	if o.Data == "" {
		return options.MissingOption("data")
	}

	var err error
	if o.Namespace == "" {
		o.Namespace, err = kubeclient.CurrentNamespace()
		if err != nil {
			return errors.Wrap(err, "failed to get current kubernetes namespace")
		}
	}
	o.DynamicClient, err = kube.LazyCreateDynamicClient(o.DynamicClient)
	if err != nil {
		return errors.Wrapf(err, "failed to create dynamic client")
	}

	ctx := context.Background()
	versionResource := o.GetGroupVersion()
	resourceName := o.ResourceKind()
	data := []byte(o.Data)
	selector := o.Selector

	if selector == "" {
		err = o.patchResource(ctx, versionResource, o.Name, data)
		if err != nil {
			return errors.Wrapf(err, "failed to patch resource %s name %s", resourceName, o.Name)
		}
	}

	resources, err := o.DynamicClient.Resource(versionResource).Namespace(o.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
	})
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Logger().Infof("could not find any resources of kind %s with selector %s", resourceName, selector)
			return nil
		}
		return errors.Wrapf(err, "failed to find resources of kind %s with selector %s", resourceName, selector)
	}
	for _, r := range resources.Items {
		name := r.GetName()
		err = o.patchResource(ctx, versionResource, name, data)
		if err != nil {
			return errors.Wrapf(err, "failed to patch resource %s name %s", resourceName, name)
		}
	}
	return nil
}

func (o *Options) ResourceKind() string {
	return strings.TrimPrefix(o.Group+"/"+o.Version+"/"+o.Kind, "/")
}

func (o *Options) GetGroupVersion() schema.GroupVersionResource {
	return schema.GroupVersionResource{Group: o.Group, Version: o.Version, Resource: o.Kind}
}

func (o *Options) patchResource(ctx context.Context, versionResource schema.GroupVersionResource, name string, data []byte) error {
	t := o.GetPatchType()
	_, err := o.DynamicClient.Resource(versionResource).Namespace(o.Namespace).Patch(ctx, name, t, data, metav1.PatchOptions{})
	if err != nil {
		return errors.Wrapf(err, "failed to patch resource %s in namespace %s using patch type %v", name, o.Namespace, t)
	}
	log.Logger().Infof("patched %s name %s/%s", info(o.ResourceKind()), info(o.Namespace), info(name))
	return nil
}

func (o *Options) GetPatchType() types.PatchType {
	switch o.Type {
	case "":
		return types.MergePatchType
	case "json":
		return types.JSONPatchType
	case "yaml":
		return types.ApplyPatchType
	default:
		return types.PatchType(o.Type)
	}
}
