package hash_test

import (
	"path/filepath"
	"testing"

	appsv1 "k8s.io/api/apps/v1"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/hash"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateAnnotatesInYamlFiles(t *testing.T) {
	sourceDir := filepath.Join("test_data", "configs")
	tmpDir := t.TempDir()

	_, ho := hash.NewCmdHashAnnotate()
	ho.SourceFiles = []string{
		filepath.Join(sourceDir, "config.yaml"),
		filepath.Join(sourceDir, "plugins.yaml"),
	}
	ho.Dir = tmpDir

	deploymentsDir := filepath.Join("test_data", "deployments")
	err := files.CopyDir(deploymentsDir, tmpDir, true)
	require.NoError(t, err, "failed to copy from %s to %s", deploymentsDir, tmpDir)

	err = ho.Run()
	assert.NoError(t, err)

	outFile := filepath.Join(tmpDir, "mydeploy.yaml")
	require.FileExists(t, outFile)

	deploy := appsv1.Deployment{}
	err = yamls.LoadFile(outFile, &deploy)
	require.NoError(t, err, "failed to load YAML file %s", outFile)

	require.NotNil(t, deploy.Annotations, "deployment has no annotations")

	value := deploy.Annotations[hash.DefaultAnnotation]
	require.NotEmpty(t, value, "no annotation %s found on file %s", hash.DefaultAnnotation, outFile)

	t.Logf("found annotation %s value: %s on file %s\n", hash.DefaultAnnotation, value, outFile)
}

func TestUpdatePodSpecAnnotatesInYamlFiles(t *testing.T) {
	sourceDir := filepath.Join("test_data", "configs")
	tmpDir := t.TempDir()

	_, ho := hash.NewCmdHashAnnotate()
	ho.SourceFiles = []string{
		filepath.Join(sourceDir, "config.yaml"),
		filepath.Join(sourceDir, "plugins.yaml"),
	}
	ho.Dir = tmpDir
	ho.PodSpec = true

	deploymentsDir := filepath.Join("test_data", "deployments")
	err := files.CopyDir(deploymentsDir, tmpDir, true)
	require.NoError(t, err, "failed to copy from %s to %s", deploymentsDir, tmpDir)

	err = ho.Run()
	assert.NoError(t, err)

	outFile := filepath.Join(tmpDir, "mydeploy.yaml")
	require.FileExists(t, outFile)

	deploy := appsv1.Deployment{}
	err = yamls.LoadFile(outFile, &deploy)
	require.NoError(t, err, "failed to load YAML file %s", outFile)

	annotations := deploy.Spec.Template.ObjectMeta.Annotations
	require.NotNil(t, annotations, "deployment has no annotations")

	value := annotations[hash.DefaultAnnotation]
	require.NotEmpty(t, value, "no annotation %s found on file %s", hash.DefaultAnnotation, outFile)

	t.Logf("found annotation %s value: %s on file %s\n", hash.DefaultAnnotation, value, outFile)
}
