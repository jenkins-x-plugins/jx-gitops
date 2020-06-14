package kyamls

import (
	"strings"

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
	kf := f.Parse()
	if len(kf.Kinds) == 0 && len(kf.KindsIgnore) == 0 {
		return nil, nil
	}
	return func(node *yaml.RNode, path string) (bool, error) {
		for _, filter := range kf.Kinds {
			if filter.Matches(node, path) {
				return true, nil
			}
		}
		for _, filter := range kf.KindsIgnore {
			if filter.Matches(node, path) {
				return false, nil
			}
		}
		if len(kf.Kinds) == 0 {
			return true, nil
		}
		return false, nil

	}, nil
}

// AddFlags add CLI flags for specifying a filter
func (f *Filter) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringArrayVarP(&f.Kinds, "kind", "k", nil, "adds Kubernetes resource kinds to filter on. For kind expressions see: https://github.com/jenkins-x/jx-gitops/tree/master/docs/kind_filters.md")
	cmd.Flags().StringArrayVarP(&f.KindsIgnore, "kind-ignore", "", nil, "adds Kubernetes resource kinds to exclude. For kind expressions see: https://github.com/jenkins-x/jx-gitops/tree/master/docs/kind_filters.md")
}

// Parse parses the filter strings
func (f *Filter) Parse() APIVersionKindsFilter {
	r := APIVersionKindsFilter{}
	for _, text := range f.Kinds {
		r.Kinds = append(r.Kinds, ParseKindFilter(text))
	}
	for _, text := range f.KindsIgnore {
		r.KindsIgnore = append(r.KindsIgnore, ParseKindFilter(text))
	}
	return r
}

// APIVersionKindsFilter a filter of kinds and/or API versions
type APIVersionKindsFilter struct {
	Kinds       []KindFilter
	KindsIgnore []KindFilter
}

// KindFilter a filter on a kind and an optional APIVersion
type KindFilter struct {
	APIVersion *string
	Kind       *string
}

// ParseKindFilter parses a kind filter
func ParseKindFilter(text string) KindFilter {
	idx := strings.LastIndex(text, "/")
	if idx >= 0 {
		apiVersion := text[0:idx]
		kind := text[idx+1:]
		if len(kind) == 0 {
			return KindFilter{
				APIVersion: &apiVersion,
			}
		}
		return KindFilter{
			APIVersion: &apiVersion,
			Kind:       &kind,
		}
	}
	return KindFilter{
		Kind: &text,
	}
}

// Matches returns true if this node matches the filter
func (f *KindFilter) Matches(node *yaml.RNode, path string) bool {
	if f.Kind != nil {
		kind := GetKind(node, path)
		if kind != *f.Kind {
			return false
		}
	}
	if f.APIVersion != nil {
		apiVersion := GetAPIVersion(node, path)
		actual := *f.APIVersion
		if apiVersion != actual && !strings.HasPrefix(apiVersion, actual) {
			return false
		}
	}
	return true
}
