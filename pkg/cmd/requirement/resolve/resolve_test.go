package resolve_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/h2non/gock"
	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/requirement/resolve"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/httphelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	expectedProject       = "myproject"
	expectedProjectNumber = "12345"
	expectedClusterName   = "cluster-something"
	expectedLocation      = "europe-west1-e"
)

var mockHeaders = map[string]string{
	"Metadata-Flavor": "Google",
}

func TestRequirementsResolve(t *testing.T) {
	// lets mock the http requests...
	client := httphelpers.GetClient()
	gock.InterceptClient(client)

	defer gock.Off()
	defer gock.RestoreClient(client)

	requests := []struct {
		path string
		body string
	}{
		{
			path: resolve.GKEPathProjectID,
			body: expectedProject,
		},
		{
			path: resolve.GKEPathProjectNumber,
			body: expectedProjectNumber,
		},
		{
			path: resolve.GKEPathClusterName,
			body: expectedClusterName,
		},
		{
			path: resolve.GKEPathClusterLocation,
			body: expectedLocation,
		},
	}

	for _, r := range requests {
		gock.New("http://metadata.google.internal").
			Get("/computeMetadata/v1/" + r.path).
			Reply(200).
			Type("text/plain").
			//SetHeaders(mockHeaders).
			BodyString(r.body)
	}

	// setup the disk
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	srcFile := filepath.Join("test_data")
	require.DirExists(t, srcFile)

	err = files.CopyDirOverwrite(srcFile, tmpDir)
	require.NoError(t, err, "failed to copy %s to %s", srcFile, tmpDir)

	// now lets run the command
	_, o := resolve.NewCmdRequirementsResolve()
	o.Dir = tmpDir
	o.NoInClusterCheck = true

	runner := &fakerunner.FakeRunner{}
	o.CommandRunner = runner.Run

	err = o.Run()
	require.NoError(t, err, "failed to run git setup")

	requirements, fileName, err := config.LoadRequirementsConfig(tmpDir, false)
	require.NoError(t, err, "failed to load requirements from %s", tmpDir)
	require.NotNil(t, requirements, "no requirements loaded in dir %s, tmpDir")

	t.Logf("modified file %s\n", fileName)

	assert.Equal(t, expectedProject, requirements.Cluster.ProjectID, "requirements.Cluster.ProjectID for file %s", fileName)
	assert.Equal(t, expectedProjectNumber, requirements.Cluster.GKEConfig.ProjectNumber, "requirements.Cluster.GKEConfig.ProjectNumber for file %s", fileName)
	assert.Equal(t, expectedClusterName, requirements.Cluster.ClusterName, "requirements.Cluster.ClusterName for file %s", fileName)
	assert.Equal(t, expectedLocation, requirements.Cluster.Zone, "requirements.Cluster.Zone for file %s", fileName)
}
