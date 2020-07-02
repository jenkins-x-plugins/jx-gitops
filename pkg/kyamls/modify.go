package kyamls

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// ModifyFiles recursively walks the given directory and modifies any suitable file
func ModifyFiles(dir string, modifyFn func(node *yaml.RNode, path string) (bool, error), filter Filter) error {
	filterFn, err := filter.ToFilterFn()
	if err != nil {
		return errors.Wrap(err, "failed to create filter")
	}

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
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

		if filterFn != nil {
			flag, err := filterFn(node, path)
			if err != nil {
				return errors.Wrapf(err, "failed to evaluate filter on file %s", path)
			}
			if !flag {
				return nil
			}
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
