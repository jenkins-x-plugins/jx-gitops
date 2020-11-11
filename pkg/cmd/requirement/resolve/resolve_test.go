package resolve_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/h2non/gock"
	"github.com/jenkins-x/jx-api/v3/pkg/config"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/requirement/resolve"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/httphelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
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

func TestRequirementsResolveGKE(t *testing.T) {
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

	srcFile := filepath.Join("test_data", "gke")
	require.DirExists(t, srcFile)

	err = files.CopyDirOverwrite(srcFile, tmpDir)
	require.NoError(t, err, "failed to copy %s to %s", srcFile, tmpDir)

	// now lets run the command
	_, o := resolve.NewCmdRequirementsResolve()
	o.Dir = tmpDir
	o.NoInClusterCheck = true

	runner := &fakerunner.FakeRunner{}
	o.CommandRunner = runner.Run
	o.KubeClient = fake.NewSimpleClientset()
	o.Namespace = "jx"

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

func TestRequirementsResolvePipelineUser(t *testing.T) {
	// setup the disk
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	srcFile := filepath.Join("test_data", "eks")
	require.DirExists(t, srcFile)

	err = files.CopyDirOverwrite(srcFile, tmpDir)
	require.NoError(t, err, "failed to copy %s to %s", srcFile, tmpDir)

	// now lets run the command
	_, o := resolve.NewCmdRequirementsResolve()
	o.Dir = tmpDir
	o.NoInClusterCheck = true

	runner := &fakerunner.FakeRunner{}
	o.CommandRunner = runner.Run

	expectedPipelineEmail := "jenkins-x@googlegroups.com"
	expectedPipelineUser := "myuser"

	ns := "jx"
	o.Namespace = ns
	o.KubeClient = fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "jx-boot",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"email":    []byte(expectedPipelineEmail),
				"username": []byte(expectedPipelineUser),
			},
		},
	)
	err = o.Run()
	require.NoError(t, err, "failed to run git setup")

	requirements, fileName, err := config.LoadRequirementsConfig(tmpDir, false)
	require.NoError(t, err, "failed to load requirements from %s", tmpDir)
	require.NotNil(t, requirements, "no requirements loaded in dir %s, tmpDir")
	pipelineUser := requirements.PipelineUser
	require.NotNil(t, pipelineUser, "no requirements.PipelineUser loaded in dir %s, tmpDir")

	t.Logf("modified file %s\n", fileName)

	assert.Equal(t, expectedPipelineUser, pipelineUser.Username, "requirements.PipelineUser.Username for file %s", fileName)
	assert.Equal(t, expectedPipelineEmail, pipelineUser.Email, "requirements.PipelineUser.Email for file %s", fileName)

	assert.NotEmpty(t, requirements.Cluster.ChartRepository, "should have requirements.Cluster.ChartRepository")
	t.Logf("have chart repository %s\n", requirements.Cluster.ChartRepository)
}
