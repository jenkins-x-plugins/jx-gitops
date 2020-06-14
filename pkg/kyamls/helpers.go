package kyamls

import (
	"strings"

	"github.com/jenkins-x/jx/pkg/log"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// GetKind finds the Kind of the node at the given path
func GetKind(node *yaml.RNode, path string) string {
	return GetStringField(node, path, "kind")
}

// GetAPIVersion finds the API Version of the node at the given path
func GetAPIVersion(node *yaml.RNode, path string) string {
	return GetStringField(node, path, "apiVersion")
}

/// GetStringField returns the given field from the node or returns a blank string if the field cannot be found
func GetStringField(node *yaml.RNode, path string, key string) string {
	kind := ""
	kindNode := node.Field(key)
	if kindNode != nil && kindNode.Value != nil {
		var err error
		kind, err = kindNode.Value.String()
		if err != nil {
			log.Logger().Warnf("failed to read field '%s'  on node for %s", key, path)
		}
	}
	return strings.TrimSpace(kind)
}

// IsClusterKind returns true if the kind is a cluster kind
func IsClusterKind(kind string) bool {
	return kind == "" || kind == "Namespace" || strings.HasPrefix(kind, "Cluster")
}
