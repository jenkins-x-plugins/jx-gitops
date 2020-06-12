package kyamls

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ModifyFiles recurses the given directory and modifies any suitable file
func ModifyFiles(dir string, modifyFn func(node *yaml.RNode, path string) (bool, error)) error {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}
		node, err := yaml.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to load file %s", path)
		}

		modified, err := modifyFn(node, path)
		if err != nil {
			return errors.Wrapf(err, "failed to modify file %s", path)
		}

		if !modified {
			return nil
		}

		err = yaml.WriteFile(node, path)
		if err != nil {
			return errors.Wrapf(err, "failed to save %s", path)
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to modify files in dir %s", dir)
	}
	return nil
}
