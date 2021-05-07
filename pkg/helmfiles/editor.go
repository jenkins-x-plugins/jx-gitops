package helmfiles

import (
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yaml2s"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/roboll/helmfile/pkg/state"
)

var (
	info = termcolor.ColorInfo
)

// Editor an editor of helmfiles
type Editor struct {
	pathToState     map[string]*state.HelmState
	namespaceToPath map[string]string
	modified        map[string]bool
	dir             string
	helmfiles       []Helmfile
}

// NewEditor creates a new editor
func NewEditor(dir string, helmfiles []Helmfile) (*Editor, error) {
	e := &Editor{
		pathToState:     map[string]*state.HelmState{},
		namespaceToPath: map[string]string{},
		modified:        map[string]bool{},
		dir:             dir,
		helmfiles:       helmfiles,
	}
	for i := range helmfiles {
		src := &helmfiles[i]
		helmState := &state.HelmState{}
		path := src.Filepath
		err := yaml2s.LoadFile(path, helmState)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load helmfile %s", path)
		}
		e.pathToState[path] = helmState
		if helmState.OverrideNamespace != "" {
			e.namespaceToPath[helmState.OverrideNamespace] = path
		}
	}
	return e, nil
}

func (e *Editor) getOrCreateState(path string) *state.HelmState {
	hf := e.pathToState[path]
	if hf == nil {
		hf = &state.HelmState{}
		e.pathToState[path] = hf
	}
	return hf
}

// ChartDetails adds a chart to the right helmfile for the given namespace
func (e *Editor) AddChart(opts *ChartDetails) error {
	ns := opts.Namespace
	if ns == "" {
		return errors.Errorf("no namespace")
	}

	path := e.namespaceToPath[ns]
	if path == "" {
		// lets create a new path
		path = filepath.Join(e.dir, ns, "helmfile.yaml")
		e.namespaceToPath[ns] = path
	}
	hf := e.getOrCreateState(path)

	rootPath := e.helmfiles[0].Filepath
	root := e.getOrCreateState(rootPath)
	rel := filepath.Join(ns, "helmfile.yaml")
	found := false
	for _, f := range root.Helmfiles {
		if f.Path == rel {
			found = true
			break
		}
	}
	if !found {
		root.Helmfiles = append(root.Helmfiles, state.SubHelmfileSpec{
			Path: rel,
		})
		e.modified[rootPath] = true
	}

	modified, err := opts.Add(hf)
	if err != nil {
		return errors.Wrapf(err, "failed to add chart")
	}
	if modified {
		e.modified[path] = true
	}
	return nil
}

// Save saves any modified files
func (e *Editor) Save() error {
	for path, f := range e.modified {
		if !f {
			continue
		}
		state := e.pathToState[path]
		if state == nil {
			return errors.Errorf("no state for path %s", path)
		}
		dir := filepath.Dir(path)
		err := os.MkdirAll(dir, files.DefaultDirWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to make dir %s", dir)
		}

		err = yaml2s.SaveFile(state, path)
		if err != nil {
			return errors.Wrapf(err, "failed to save file %s", path)
		}

		log.Logger().Infof("saved %s", info(path))
	}
	return nil
}
