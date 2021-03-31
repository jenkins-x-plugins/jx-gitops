package copy

import (
	"context"
	"fmt"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
)

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Copies kubernetes resources (by default confimaps) from a namespace to the current namespace
`)

	cmdExample = templates.Examples(`
		# copies the config map with named beer to a namespace
		%s copy --name beer --to=foo

		# copies config maps with a selector to a namespace
		%s copy -l mylabel=something --to=foo

		# copies resources matching a selector and kind
		%s copy --kind ingresses -l mylabel=something --to=foo
	`)
)

// Options the options for the command
type Options struct {
	Namespace       string
	ToNamespace     string
	Group           string
	Version         string
	Kind            string
	Selector        string
	Name            string
	Query           string
	CreateNamespace bool
	Count           int
	DynamicClient   dynamic.Interface
	KubeClient      kubernetes.Interface
}

// NewCmdCopy creates a command object for the command
func NewCmdCopy() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "copy",
		Short:   "Copies resources (by default confimaps) with the given selector or name from a source namespace to a destination namespace",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Namespace, "ns", "n", "", "the namespace to find the resources to copy. Defaults to the current namespace")
	cmd.Flags().StringVarP(&o.ToNamespace, "to", "t", "", "the namespace to copy the secrets to")
	cmd.Flags().StringVarP(&o.Group, "group", "g", "", "the API group such as 'apps' for Deployemnts")
	cmd.Flags().StringVarP(&o.Version, "version", "", "v1", "the API version of the resources to copy")
	cmd.Flags().StringVarP(&o.Kind, "kind", "k", "configmaps", "the kind name")
	cmd.Flags().StringVarP(&o.Selector, "selector", "l", "", "the label selector to find the resources to copy")
	cmd.Flags().StringVarP(&o.Name, "name", "", "", "the name of the resource to copy instead of a selector")
	cmd.Flags().BoolVarP(&o.CreateNamespace, "create-namespace", "", false, "create the to Namespace if it does not already exist")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	if o.ToNamespace == "" {
		return options.MissingOption("to")
	}
	if o.Selector == "" && o.Name == "" {
		return options.MissingOption("selector")
	}

	var err error
	o.KubeClient, o.Namespace, err = kube.LazyCreateKubeClientAndNamespace(o.KubeClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to create kube client")
	}
	o.DynamicClient, err = kube.LazyCreateDynamicClient(o.DynamicClient)
	if err != nil {
		return errors.Wrapf(err, "failed to create dynamic client")
	}
	if o.CreateNamespace {
		err = jxenv.EnsureNamespaceCreated(o.KubeClient, o.ToNamespace, nil, nil)
		if err != nil {
			return errors.Wrapf(err, "failed to create namespace %s", o.ToNamespace)
		}
	}

	ns := o.Namespace
	selector := o.Selector
	fieldSelector := ""
	if o.Name != "" {
		fieldSelector = "metadata.name=" + o.Name
	}

	versionResource := schema.GroupVersionResource{Group: o.Group, Version: o.Version, Resource: o.Kind}

	resourceName := strings.TrimPrefix(o.Group+"/"+o.Version+"/"+o.Kind, "/")

	ctx := context.Background()
	resources, err := o.DynamicClient.Resource(versionResource).Namespace(o.Namespace).List(ctx, metav1.ListOptions{
		LabelSelector: selector,
		FieldSelector: fieldSelector,
	})

	query := ""
	if selector != "" {
		query = "selector " + selector
	} else {
		query = "fieldSelector " + fieldSelector
	}
	o.Query = query

	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Logger().Warnf("no %s in namespace %s with query %s", resourceName, ns, query)
			return nil
		}
		return errors.Wrapf(err, "failed to find %s in namespace %s with query %s", resourceName, ns, query)
	}
	for i := range resources.Items {
		r := resources.Items[i]

		r.SetNamespace(o.ToNamespace)
		r.SetResourceVersion("")
		r.SetUID(types.UID(""))

		_, err = o.DynamicClient.Resource(versionResource).Namespace(o.ToNamespace).Create(ctx, &r, metav1.CreateOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to copy %s %s to namespace %s", resourceName, r.GetName(), o.ToNamespace)
		}

		log.Logger().Infof("copied copied %s %s to namespace %s", resourceName, r.GetName(), o.ToNamespace)
		o.Count++
	}
	if o.Count == 0 {
		log.Logger().Infof("did not find any %s resources in namespace %s matching query %s", resourceName, o.Namespace, query)
	}
	return nil
}
