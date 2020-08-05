package kyamls

import (
	"strings"

	"github.com/jenkins-x/jx-logging/pkg/log"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var quotes = []string{"'", "\""}

// GetKind finds the Kind of the node at the given path
func GetKind(node *yaml.RNode, path string) string {
	return GetStringField(node, path, "kind")
}

// GetAPIVersion finds the API Version of the node at the given path
func GetAPIVersion(node *yaml.RNode, path string) string {
	return GetStringField(node, path, "apiVersion")
}

// GetName returns the name from the metadata
func GetName(node *yaml.RNode, path string) string {
	return GetStringField(node, path, "metadata", "name")
}

// GetNamespace returns the namespace from the metadata
func GetNamespace(node *yaml.RNode, path string) string {
	return GetStringField(node, path, "metadata", "namespace")
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
	return TrimSpaceAndQuotes(answer)
}

// TrimSpaceAndQuotes trims any whitespace and quotes around a value
func TrimSpaceAndQuotes(answer string) string {
	text := strings.TrimSpace(answer)
	for _, q := range quotes {
		if strings.HasPrefix(text, q) && strings.HasSuffix(text, q) {
			return strings.TrimPrefix(strings.TrimSuffix(text, q), q)
		}
	}
	return text
}

// IsClusterKind returns true if the kind is a cluster kind
func IsClusterKind(kind string) bool {
	return kind == "" || kind == "CustomResourceDefinition" || kind == "Namespace" || strings.HasPrefix(kind, "Cluster")
}

// JSONPath returns the fields separated by dots
func JSONPath(fields ...string) string {
	return strings.Join(fields, ".")
}
