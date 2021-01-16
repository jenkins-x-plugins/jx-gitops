package helmfiles

import (
	"os"
	"path/filepath"
	"strings"

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
func GatherHelmfiles(helmfile string) ([]Helmfile, error) {
	helmState := state.HelmState{}
	err := yaml2s.LoadFile(helmfile, &helmState)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load helmfile %s", helmfile)
	}

	helmfiles := []Helmfile{
		{helmfile, ""},
	}
	parentHelmfileDir := filepath.Dir(helmfile)
	for _, nested := range helmState.Helmfiles {
		nestedHelmfileDepth := len(strings.Split(filepath.Dir(nested.Path), pathSeparator))
		relativePath := strings.Repeat("../", nestedHelmfileDepth)

		helmfiles = append(helmfiles, Helmfile{filepath.Join(parentHelmfileDir, nested.Path), relativePath})
	}
	return helmfiles, nil
}
