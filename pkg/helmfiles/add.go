package helmfiles

import (
	"fmt"
	"strings"

	"github.com/helmfile/helmfile/pkg/state"
	"github.com/jenkins-x/jx-helpers/v3/pkg/versionstream"
)

// ChartDetails the chart options when adding/updating charts
type ChartDetails struct {
	Namespace   string
	Chart       string
	Repository  string
	Version     string
	ReleaseName string
	Values      []string
	UpdateOnly  bool
	Prefixes    *versionstream.RepositoryPrefixes
}

// NewChartDetails return new add chart options from an existing release
func NewChartDetails(helmState *state.HelmState, rel *state.ReleaseSpec, prefixes *versionstream.RepositoryPrefixes) *ChartDetails {
	a := &ChartDetails{
		Namespace:   rel.Namespace,
		Chart:       rel.Chart,
		Version:     rel.Version,
		ReleaseName: rel.Name,
		Values:      nil,
		Prefixes:    prefixes,
	}
	if a.Namespace == "" {
		a.Namespace = helmState.OverrideNamespace
	}
	prefix, _ := SpitChartName(a.Chart)
	if a.Repository == "" && prefix != "" {
		for k := range helmState.Repositories {
			r := helmState.Repositories[k]
			if r.Name == prefix {
				a.Repository = r.URL
			}
		}
	}
	return a
}

// SpitChartName splits the chart name into prefix and local name
func SpitChartName(name string) (string, string) {
	prefix := ""
	local := name
	parts := strings.Split(name, "/")
	if len(parts) > 1 {
		prefix = parts[0]
		local = parts[1]
	}
	return prefix, local
}

// String returns the string representation of the chart options
func (o *ChartDetails) String() string {
	return o.Chart + " in namespace " + o.Namespace
}

// Add adds or updates the chart details in the helm state
func (o *ChartDetails) Add(helmState *state.HelmState) (bool, error) {
	modified := false
	found := false
	prefix, localName := SpitChartName(o.Chart)
	if o.ReleaseName == "" {
		o.ReleaseName = localName
	}

	// lets resolve the chart prefix from a local repository from the file or from a
	// prefix in the versions stream
	_, err := AddRepository(helmState, prefix, o.Repository, o.Prefixes)
	if err != nil {
		return false, fmt.Errorf("failed to add repository for release %s: %w", o.ReleaseName, err)
	}


	// lets only set the namespace if its different to the default to keep the helmfiles DRY
	namespace := o.Namespace
	if namespace == helmState.OverrideNamespace {
		namespace = ""
	}
	for i := range helmState.Releases {
		release := &helmState.Releases[i]
		if release.Chart == o.Chart && release.Name == o.ReleaseName {
			found = true
			if release.Namespace != "" && release.Namespace != namespace {
				release.Namespace = namespace
				modified = true
			}
			if release.Version != o.Version && o.Version != "" {
				release.Version = o.Version
				modified = true
			}

			// lets add any missing values
			for _, v := range o.Values {
				foundValue := false
				for j := range release.Values {
					if release.Values[j] == v {
						foundValue = true
						break
					}
				}
				if !foundValue {
					release.Values = append(release.Values, v)
					modified = true
				}
			}
			break
		}
	}
	if !found && !o.UpdateOnly {
		release := state.ReleaseSpec{
			Chart:     o.Chart,
			Version:   o.Version,
			Name:      o.ReleaseName,
			Namespace: namespace,
		}
		for _, v := range o.Values {
			release.Values = append(release.Values, v)
		}
		helmState.Releases = append(helmState.Releases, release)
		modified = true
	}
	return modified, nil
}

// Delete removes the releases for the given details from the given helm state
func (o *ChartDetails) Delete(helmState *state.HelmState) (bool, error) {
	modified := false
	last := len(helmState.Releases) - 1
	for i := last; i >= 0; i-- {
		release := &helmState.Releases[i]
		if MatchesChartName(release.Chart, o.Chart) && (o.ReleaseName == "" || release.Name == o.ReleaseName) {
			r2 := helmState.Releases[0:i]
			if i < last {
				r2 = append(r2, helmState.Releases[i+1:]...)
			}
			helmState.Releases = r2
			modified = true
		}
	}
	return modified, nil
}

// MatchesChartName if name has a prefix then match on prefix and name otherwise just match on the local name only
func MatchesChartName(releaseChart, name string) bool {
	if strings.Contains(name, "/") {
		return releaseChart == name
	}
	_, localName := SpitChartName(releaseChart)
	return localName == name
}
