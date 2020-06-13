package kyamls

import (
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// Filter for filtering
type Filter struct {
	Kinds       []string
	KindsIgnore []string
}

// ToFilterFn creates a filter function
func (f *Filter) ToFilterFn() (func(node *yaml.RNode, path string) (bool, error), error) {
	if len(f.Kinds) == 0 && len(f.KindsIgnore) == 0 {
		return nil, nil
	}

	return func(node *yaml.RNode, path string) (bool, error) {
		kind := GetKind(node, path)
		if matches(kind, f.KindsIgnore) {
			return false, nil
		}
		if matches(kind, f.Kinds) {
			return true, nil
		}
		return false, nil

	}, nil
}

// AddFlags add CLI flags for specifying a filter
func (f *Filter) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringArrayVarP(&f.Kinds, "kind", "k", nil, "adds Kubernetes resource kinds to filter on")
	cmd.Flags().StringArrayVarP(&f.KindsIgnore, "kind-ignore", "", nil, "adds Kubernetes resource kinds to exclude")
}

func matches(kind string, kinds []string) bool {
	for _, k := range kinds {
		if kind == k {
			return true
		}
	}
	return false
}
