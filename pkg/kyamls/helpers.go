package kyamls

import (
	"strings"

	"github.com/jenkins-x/jx/pkg/log"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// GetKind finds the Kind of the node at the given path
func GetKind(node *yaml.RNode, path string) string {
	kind := ""
	kindNode := node.Field("kind")
	if kindNode != nil && kindNode.Value != nil {
		var err error
		kind, err = kindNode.Value.String()
		if err != nil {
			log.Logger().Warnf("failed to read kind on node for %s", path)
		}
	}
	return strings.TrimSpace(kind)
}

// IsClusterKind returns true if the kind is a cluster kind
func IsClusterKind(kind string) bool {
	return kind == "" || kind == "Namespace" || strings.HasPrefix(kind, "Cluster")
}
