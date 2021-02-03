package helmfiles

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yaml2s"
	"github.com/pkg/errors"
	"github.com/roboll/helmfile/pkg/state"
)

type Helmfile struct {
	Filepath           string
	RelativePathToRoot string
}

var (
	pathSeparator = string(os.PathSeparator)
)

// GatherHelmfiles gathers the helmfiles from the given file
func GatherHelmfiles(helmfile, dir string) ([]Helmfile, error) {
	parentHelmfileDir := filepath.Dir(helmfile)

	// we need to check if the main helmfile itself is in a subdirectory, if it is then we need to add that to any
	// nested subfolders we find so we can correctly reference version stream files
	parentHelmfileDepth := 0
	if strings.Contains(helmfile, pathSeparator) {
		parentHelmfileDepth = len(strings.Split(parentHelmfileDir, pathSeparator))
	}

	helmfile = filepath.Join(dir, helmfile)
	helmState := state.HelmState{}
	err := yaml2s.LoadFile(helmfile, &helmState)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load helmfile %s", helmfile)
	}

	helmfiles := []Helmfile{
		{helmfile, ""},
	}
	parentHelmfileDir = filepath.Dir(helmfile)

	for _, nested := range helmState.Helmfiles {
		nestedHelmfileDepth := len(strings.Split(filepath.Dir(nested.Path), pathSeparator))
		relativePath := strings.Repeat("../", parentHelmfileDepth+nestedHelmfileDepth)

		fileLocation := filepath.Join(parentHelmfileDir, nested.Path)
		exists, err := files.FileExists(fileLocation)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to check for nested helmfile %s", fileLocation)
		}
		if !exists {
			return nil, fmt.Errorf("failed to find nested helmfile %s", fileLocation)
		}
		helmfiles = append(helmfiles, Helmfile{fileLocation, relativePath})
	}
	return helmfiles, nil
}
