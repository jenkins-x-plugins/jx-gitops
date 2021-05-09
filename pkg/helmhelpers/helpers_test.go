package helmhelpers_test

import (
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/helmhelpers"
	"github.com/roboll/helmfile/pkg/state"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindClusterLocalRepositoryURLs(t *testing.T) {
	repos := []state.RepositorySpec{
		{
			Name: "jx",
			URL:  "https://storage.googleapis.com/chartmuseum.jenkins-x.io",
		},
		{
			Name: "bucketrepo",
			URL:  "http://bucketrepo/bucketrepo/charts/",
		},
		{
			Name: "concise-chart-museum",
			URL:  "http://jenkins-x-chartmuseum:8080",
		},
		{
			Name: "full-chart-museum",
			URL:  "http://jenkins-x-chartmuseum.jx.svc.cluster.local:8080",
		},
	}

	localRepos, err := helmhelpers.FindClusterLocalRepositoryURLs(repos)
	require.NoError(t, err, "failed to find local cluster repos")

	t.Logf("found local repos %s\n", localRepos)

	expected := []string{"http://bucketrepo/bucketrepo/charts/", "http://jenkins-x-chartmuseum:8080", "http://jenkins-x-chartmuseum.jx.svc.cluster.local:8080"}
	notExpected := []string{"https://storage.googleapis.com/chartmuseum.jenkins-x.io"}

	for _, name := range expected {
		assert.Contains(t, localRepos, name, "local repo %s", name)
	}
	for _, name := range notExpected {
		assert.NotContains(t, localRepos, name, "local repo %s", name)
	}
}
