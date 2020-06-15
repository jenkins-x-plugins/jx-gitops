package kyamls

import (
	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// SetStringValue sets the string value at the given path
func SetStringValue(node *yaml.RNode, path string, value string, fields ...string) error {
	err := node.PipeE(yaml.LookupCreate(yaml.ScalarNode, fields...), yaml.FieldSetter{StringValue: value})
	if err != nil {
		return errors.Wrapf(err, "failed to set field %s to %s at path %s", JSONPath(fields...), value, path)
	}
	return nil
}
