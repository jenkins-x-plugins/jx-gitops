package kustomizes

import (
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/pkg/errors"
	"sigs.k8s.io/kustomize/api/types"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

// LazyCreate lazily creates the kustomization configuration
func LazyCreate(k *types.Kustomization) *types.Kustomization {
	if k == nil {
		k = &types.Kustomization{}
	}
	k.FixKustomization()
	return k
}

// LoadKustomization loads the kustomization yaml file from the given directory
func LoadKustomization(dir string) (*types.Kustomization, error) {
	fileName := filepath.Join(dir, "kustomization.yaml")
	exists, err := files.FileExists(fileName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to check if file exists %s", fileName)
	}

	answer := &types.Kustomization{}
	answer.FixKustomization()

	if !exists {
		return answer, nil
	}
	data, err := os.ReadFile(fileName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load file %s", fileName)
	}
	err = yaml.Unmarshal(data, answer)
	if err != nil {
		return nil, errors.Wrapf(err, "failed parse YAML file %s", fileName)
	}
	return answer, nil
}

// SaveKustomization saves the kustomisation file in the given directory
func SaveKustomization(kustomization *types.Kustomization, dir string) error {
	data, err := yaml.Marshal(kustomization)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal Kustomization")
	}
	fileName := filepath.Join(dir, "kustomization.yaml")
	err = os.WriteFile(fileName, data, files.DefaultFileWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed write file %s", fileName)
	}
	return nil
}
