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
	if helmfile == "" {
		helmfile = "helmfile.yaml"
	}
	baseParentHelmfileDir := filepath.Dir(helmfile)

	// we need to check if the main helmfile itself is in a subdirectory, if it is then we need to add that to any
	// nested subfolders we find so we can correctly reference version stream files
	parentHelmfileDepth := 0
	if strings.Contains(helmfile, pathSeparator) {
		parentHelmfileDepth = len(strings.Split(baseParentHelmfileDir, pathSeparator))
	}

	helmfile = filepath.Join(dir, helmfile)
	helmState := state.HelmState{}
	err := yaml2s.LoadFile(helmfile, &helmState)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to load helmfile %s", helmfile)
	}

	relativePath := strings.Repeat("../", parentHelmfileDepth)

	helmfiles := []Helmfile{
		{helmfile, relativePath},
	}
	parentHelmfileDir := filepath.Dir(helmfile)

	for _, nested := range helmState.Helmfiles {
		// lets ignore remote helmfiles
		if strings.HasPrefix(nested.Path, "git::") {
			continue
		}

		// recursively gather nested helmfiles including their relative path so we can add correct location to version stream values files
		// note: unit tests cover this as it is a complex function however they set a test dir with files copied into it,
		// when running the resolve command for real the dir is '.' and therefore can take a slightly different route through this
		// func.  If you change this then its also worth giving a manual test of jx giotops helmfile resolve on a nested helmfile
		// and make sure the relative path to version stream values remain the same.
		nestedHelmfile, err := GatherHelmfiles(filepath.Join(baseParentHelmfileDir, nested.Path), dir)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get nested helmnfiles %s in %s", nested.Path, dir)
		}
		for _, h := range nestedHelmfile {
			helmfiles = append(helmfiles, h)
		}

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
