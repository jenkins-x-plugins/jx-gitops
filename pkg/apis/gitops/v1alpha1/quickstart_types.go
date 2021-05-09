package v1alpha1

import (
	"fmt"
	"path/filepath"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/matcher"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	// QuickstartsFileName default name of the source repository configuration
	QuickstartsFileName = "quickstarts.yaml"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Quickstarts represents a collection quickstart project
//
// +k8s:openapi-gen=true
type Quickstarts struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// Spec holds the specified quicksatrt configuration
	// +optional
	Spec QuickstartsSpec `json:"spec"`
}

// QuickstartsList contains a list of Quickstarts
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type QuickstartsList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Quickstarts `json:"items"`
}

// QuickstartsSpec defines the desired state of Quickstarts.
type QuickstartsSpec struct {
	// Quickstarts custom quickstarts to include
	Quickstarts []QuickstartSource `json:"quickstarts,omitempty"`

	// DefaultOwner the default owner if not specfied
	DefaultOwner string `json:"defaultOwner,omitempty"`

	// Imports import quickstarts from the version stream
	Imports []QuickstartImport `json:"imports,omitempty"`
}

// QuickstartSource the source of a quickstart
type QuickstartSource struct {
	ID             string
	Owner          string
	Name           string
	Version        string
	Language       string
	Framework      string
	Tags           []string
	DownloadZipURL string
	GitServer      string
	GitKind        string
}

// DefaultValues defaults any missing values
func (qs *QuickstartsSpec) DefaultValues(q *QuickstartSource) {
	if qs.DefaultOwner == "" {
		qs.DefaultOwner = "jenkins-x-quickstarts"
	}
	if q.Owner == "" {
		q.Owner = qs.DefaultOwner
	}
	if q.ID == "" {
		q.ID = fmt.Sprintf("%s/%s", q.Owner, q.Name)
	}
	if q.DownloadZipURL == "" {
		q.DownloadZipURL = fmt.Sprintf("https://codeload.github.com/%s/%s/zip/master", q.Owner, q.Name)
	}
}

// LoadImports loads the imported quickstarts
func (qs *QuickstartsSpec) LoadImports(i *QuickstartImport, matcher func(source *QuickstartSource) bool, dir string) ([]QuickstartSource, error) {
	if i.File == "" {
		return nil, errors.Errorf("missing file name on import")
	}
	fileName := filepath.Join(dir, i.File)
	exists, err := files.FileExists(fileName)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to verify file exists %s", fileName)
	}
	if !exists {
		return nil, errors.Errorf("imported file does not exist %s", fileName)
	}

	quickstarts := &Quickstarts{}
	err = yamls.LoadFile(fileName, quickstarts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse %s", fileName)
	}

	var answer []QuickstartSource
	for i := range quickstarts.Spec.Quickstarts {
		q := &quickstarts.Spec.Quickstarts[i]
		quickstarts.Spec.DefaultValues(q)
		if matcher(q) {
			answer = append(answer, *q)
		}
	}
	return answer, nil
}

// QuickstartImport imports quickstats from another folder (such as from the shared version stream)
type QuickstartImport struct {
	// File file name relative to the root directory to load
	File     string   `json:"file,omitempty"`
	Include  []string `json:"includes,omitempty"`
	Excludes []string `json:"excludes,omitempty"`
}

// Matcher returns a matcher for the given import
func (i *QuickstartImport) Matcher() (func(source *QuickstartSource) bool, error) {
	matcher := matcher.Matcher{}
	var err error
	matcher.Includes, err = matcher.ToRegexs(i.Include)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create include regex")
	}
	matcher.Excludes, err = matcher.ToRegexs(i.Excludes)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create exclude regex")
	}
	return func(q *QuickstartSource) bool {
		return matcher.Matches(q.ID)
	}, nil
}
