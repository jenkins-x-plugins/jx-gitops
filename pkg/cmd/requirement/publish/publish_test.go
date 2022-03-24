package publish_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/requirement/publish"
	v1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/stretchr/testify/require"
)

func TestRequirementsPublish(t *testing.T) {
	tmpDir := t.TempDir()

	err := files.CopyDirOverwrite("test_data", tmpDir)
	require.NoError(t, err, "failed to copy %s to %s", "test_data", tmpDir)

	_, o := publish.NewCmdRequirementsPublish()

	o.Dir = tmpDir
	err = o.Run()
	require.NoError(t, err, "failed to run")

	expectedEnvFile := filepath.Join(tmpDir, "config-root", "namespaces", "jx", "jxboot-helmfile-resources", "dev-environment.yaml")

	require.FileExists(t, expectedEnvFile, "should have an environment file")

	env := &v1.Environment{}
	err = yamls.LoadFile(expectedEnvFile, env)
	require.NoError(t, err, "failed to load Environment from %s", expectedEnvFile)
}
