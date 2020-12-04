package move_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/move"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateNamespaceInYamlFiles(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	_, o := move.NewCmdHelmfileMove()

	t.Logf("generating output to %s\n", tmpDir)

	o.Dir = filepath.Join("test_data", "output")
	o.OutputDir = tmpDir

	err = o.Run()
	require.NoError(t, err, "failed to run helmfile move")

	expectedFiles := []string{
		filepath.Join(tmpDir, "customresourcedefinitions", "jx", "lighthouse", "lighthousejobs.lighthouse.jenkins.io-crd.yaml"),
		filepath.Join(tmpDir, "cluster", "nginx", "nginx-ingress", "nginx-ingress-clusterrole.yaml"),
		filepath.Join(tmpDir, "namespaces", "jx", "lighthouse", "lighthouse-foghorn-deploy.yaml"),
	}
	for _, ef := range expectedFiles {
		assert.FileExists(t, ef)
		t.Logf("generated expected file %s\n", ef)
	}
}
