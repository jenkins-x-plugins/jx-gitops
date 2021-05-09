package quickstarthelpers

import (
	"path/filepath"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/pkg/errors"
)

// LoadQuickstarts loads the quickstarts configuration if it exists
func LoadQuickstarts(dir string) (*v1alpha1.Quickstarts, string, error) {
	fileName := filepath.Join(dir, "extensions", v1alpha1.QuickstartsFileName)
	exists, err := files.FileExists(fileName)
	if err != nil {
		return nil, fileName, errors.Wrapf(err, "failed to check if file exists %s", fileName)
	}
	qs := &v1alpha1.Quickstarts{}
	if exists {
		err = yamls.LoadFile(fileName, qs)
		if err != nil {
			return nil, fileName, errors.Wrapf(err, "failed to load Quickstarts file %s", fileName)
		}
	}
	return qs, fileName, nil
}
