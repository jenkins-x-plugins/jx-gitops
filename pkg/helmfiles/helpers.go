package helmfiles

import (
	"fmt"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/versionstreamer"
	"github.com/jenkins-x/jx-helpers/v3/pkg/versionstream"
	"os"
	"path/filepath"
	"strings"

	"github.com/helmfile/helmfile/pkg/state"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yaml2s"
	"github.com/pkg/errors"
)

type Helmfile struct {
	Filepath           string
	RelativePathToRoot string
}

var pathSeparator = string(os.PathSeparator)

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
		helmfiles = append(helmfiles, nestedHelmfile...)
	}
	return helmfiles, nil
}

// AddRepository ensures that the helm repository for the prefix exists in the helmstate.
// For it to succeed either repositoryUrl needs to be set or the prefix exists in prefixes.
func AddRepository(helmState *state.HelmState, prefix string, repositoryUrl string, prefixes *versionstream.RepositoryPrefixes) (string, error) {
	// lets resolve the chart prefix from a local repository from the file or from a
	// prefix in the versions stream
	var oci bool
	if prefix != "" && repositoryUrl == "" {
		for k := range helmState.Repositories {
			r := helmState.Repositories[k]
			if r.Name == prefix {
				repositoryUrl = r.URL
				oci = r.OCI
			}
		}
	}
	var err error
	if repositoryUrl == "" && prefix != "" {
		repositoryUrl, err = versionstreamer.MatchRepositoryPrefix(prefixes, prefix)
		if err != nil {
			return "", errors.Wrapf(err, "failed to match prefix %s with repositories from versionstream", prefix)
		}
	}
	if repositoryUrl == "" && prefix != "" {
		return "", errors.Wrapf(err, "failed to find repository URL, not defined in helmfile.yaml or versionstream")
	}
	if repositoryUrl != "" && prefix != "" {
		ociPrefix := strings.HasPrefix(repositoryUrl, "oci://")
		if ociPrefix {
			repositoryUrl = repositoryUrl[len("oci://"):]
			oci = true
		}
		// lets ensure we've got a repository for this URL in the apps file
		found := false
		for k := range helmState.Repositories {
			r := helmState.Repositories[k]
			if r.Name == prefix {
				if r.URL != repositoryUrl {
					return "",
						fmt.Errorf("release has prefix %s for repository URL %s which is also mapped to prefix %s",
							prefix, r.URL, r.Name)
				}
				found = true
				break
			}
		}
		if !found {
			helmState.Repositories = append(helmState.Repositories, state.RepositorySpec{
				Name: prefix,
				URL:  repositoryUrl,
				OCI:  oci,
			})
		}
	}
	return repositoryUrl, nil
}

