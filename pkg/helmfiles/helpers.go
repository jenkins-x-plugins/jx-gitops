package helmfiles

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/goccy/go-yaml"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/versionstreamer"
	"github.com/jenkins-x/jx-helpers/v3/pkg/versionstream"

	"github.com/helmfile/helmfile/pkg/state"
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

	file, err := os.Open(helmfile)
	if err != nil {
		return nil, err
	}
	relativePath := strings.Repeat("../", parentHelmfileDepth)

	helmfiles := []Helmfile{
		{helmfile, relativePath},
	}

	dec := yaml.NewDecoder(file)
	for {
		helmState := state.HelmState{}
		if err := dec.Decode(&helmState); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, fmt.Errorf("failed to load helmfile %s: %w", helmfile, err)
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
	}
	return helmfiles, nil
}

// AddRepository ensures that the helm repository for the prefix exists in the helmstate.
// For it to succeed either repositoryUrl needs to be set or the prefix exists in prefixes.
func AddRepository(helmState *state.HelmState, prefix, repositoryURL string, prefixes *versionstream.RepositoryPrefixes) (string, error) {
	// lets resolve the chart prefix from a local repository from the file or from a
	// prefix in the versions stream
	var oci bool
	if prefix != "" && repositoryURL == "" {
		for k := range helmState.Repositories {
			r := helmState.Repositories[k]
			if r.Name == prefix {
				repositoryURL = r.URL
				oci = r.OCI
			}
		}
	}
	var err error
	if repositoryURL == "" && prefix != "" {
		repositoryURL, err = versionstreamer.MatchRepositoryPrefix(prefixes, prefix)
		if err != nil {
			return "", errors.Wrapf(err, "failed to match prefix %s with repositories from versionstream", prefix)
		}
	}
	if repositoryURL == "" && prefix != "" {
		return "", errors.Wrapf(err, "failed to find repository URL, not defined in helmfile.yaml or versionstream")
	}
	if repositoryURL != "" && prefix != "" {
		ociPrefix := strings.HasPrefix(repositoryURL, "oci://")
		if ociPrefix {
			repositoryURL = repositoryURL[len("oci://"):]
			oci = true
		}
		// lets ensure we've got a repository for this URL in the apps file
		found := false
		for k := range helmState.Repositories {
			r := helmState.Repositories[k]
			if r.Name == prefix {
				if r.URL != repositoryURL {
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
				URL:  repositoryURL,
				OCI:  oci,
			})
		}
	}
	return repositoryURL, nil
}

// LoadHelmfile loads helmfile from a path
func LoadHelmfile(path string) ([]*state.HelmState, error) {
	var helmStates []*state.HelmState

	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	dec := yaml.NewDecoder(file)
	for {
		helmState := state.HelmState{}
		if err := dec.Decode(&helmState); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		helmStates = append(helmStates, &helmState)
	}
	err = file.Close()
	if err != nil {
		return nil, err
	}

	return helmStates, nil
}

// SaveHelmfile saves helmfile to a path
func SaveHelmfile(path string, helmStates []*state.HelmState) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	env := yaml.NewEncoder(
		file,
		yaml.OmitEmpty(),
		yaml.UseLiteralStyleIfMultiline(true),
		yaml.UseSingleQuote(true),
	)
	for i := range helmStates {
		err := env.Encode(*helmStates[i])
		if err != nil {
			return fmt.Errorf("failed to save file %s: %w", path, err)
		}
	}
	err = file.Sync()
	if err != nil {
		return err
	}
	err = file.Close()
	if err != nil {
		return fmt.Errorf("failed to save file %s: %w", path, err)
	}

	return nil
}
