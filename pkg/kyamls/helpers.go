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
func GetStringField(node *yaml.RNode, path string, fields ...string) string {
	answer := ""
	valueNode, err := node.Pipe(yaml.Lookup(fields...))
	if err != nil {
		log.Logger().Warnf("failed to read field %s for path %s", JSONPath(fields...), path)
	}
	if valueNode != nil {
		var err error
		answer, err = valueNode.String()
		if err != nil {
			log.Logger().Warnf("failed to get string value of field %s for path %s", JSONPath(fields...), path)
		}
	}
	return strings.TrimSpace(answer)
}

// IsClusterKind returns true if the kind is a cluster kind
func IsClusterKind(kind string) bool {
	return kind == "" || kind == "Namespace" || strings.HasPrefix(kind, "Cluster")
}

// JSONPath returns the fields separated by dots
func JSONPath(fields ...string) string {
	return strings.Join(fields, ".")
}
