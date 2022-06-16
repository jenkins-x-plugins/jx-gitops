package releasereport

import (
	"fmt"

	"helm.sh/helm/v3/pkg/chart"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NamespaceReleases the releases for a namespace
type NamespaceReleases struct {
	Path      string         `json:"path,omitempty"`
	Namespace string         `json:"namespace,omitempty"`
	Releases  []*ReleaseInfo `json:"releases,omitempty"`
}

// ReleaseInfo information about the release
type ReleaseInfo struct {
	chart.Metadata

	// ReleaseName is the name of the helm release
	ReleaseName string `json:"releaseName,omitempty"`

	// FirstDeployed is when the chart version was first deployed.
	FirstDeployed *metav1.Time `json:"firstDeployed,omitempty"`
	// LastDeployed is when the release was last deployed.
	LastDeployed *metav1.Time `json:"lastDeployed,omitempty"`

	// RepositoryName the chart repository name used in the fully qualified chart name
	RepositoryName string `json:"repositoryName,omitempty"`
	// RepositoryURL the chart repository URL
	RepositoryURL string `json:"repositoryUrl,omitempty"`
	// ApplicationURL the ingress URL for the application if available
	ApplicationURL string `json:"applicationUrl,omitempty"`
	// LogsURL the URL to browse the application logs if available
	LogsURL string `json:"logsUrl,omitempty"`

	// ResourcesPath the relative path to the kubernetes resources
	ResourcesPath string `json:"resourcePath,omitempty"`

	// Ingresses the ingress URLs
	Ingresses []IngressInfo `json:"ingresses,omitempty"`
}

// IngressInfo details of an ingress
type IngressInfo struct {
	Name string `json:"name,omitempty"`
	URL  string `json:"url,omitempty"`
}

func (i *ReleaseInfo) String() string {
	answer := fmt.Sprintf("%s version: %s", i.Name, i.Version)
	if i.Home != "" {
		answer += " " + i.Home
	}
	return answer
}
