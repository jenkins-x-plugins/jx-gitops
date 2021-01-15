package report

import (
	"fmt"

	"helm.sh/helm/v3/pkg/chart"
)

type NamespaceCharts struct {
	Path      string       `json:"path,omitempty"`
	Namespace string       `json:"namespace,omitempty"`
	Charts    []*ChartInfo `json:"charts,omitempty"`
}

type ChartInfo struct {
	chart.Metadata
	RepositoryName string `json:"repositoryName,omitempty"`
	RepositoryURL  string `json:"repositoryUrl,omitempty"`
	ApplicationURL string `json:"applicationUrl,omitempty"`
}

func (i *ChartInfo) String() string {
	return fmt.Sprintf("%s version: %s icon: %s", i.Name, i.Version, i.Icon)
}

func (i *ChartInfo) handleChartMetadata(manifest *chart.Metadata) {
	if i.Description == "" {
		i.Description = manifest.Description
	}
	if i.Home == "" {
		i.Home = manifest.Home
	}
	if i.Icon == "" {
		i.Icon = manifest.Icon
	}
}
