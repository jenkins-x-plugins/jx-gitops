package merge_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/requirement/merge"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/testhelpers"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestRequirementsMergeFile(t *testing.T) {
	// setup the disk
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	srcDir := filepath.Join("test_data")
	require.DirExists(t, srcDir)

	err = files.CopyDirOverwrite(srcDir, tmpDir)
	require.NoError(t, err, "failed to copy %s to %s", srcDir, tmpDir)

	// now lets run the command
	_, o := merge.NewCmdRequirementsMerge()
	o.Dir = tmpDir
	o.File = filepath.Join(tmpDir, "changes.yml")

	t.Logf("merging requirements in dir %s\n", tmpDir)

	err = o.Run()
	require.NoError(t, err, "failed to run merge")

	testhelpers.AssertTextFilesEqual(t, filepath.Join(tmpDir, "expected.yml"), filepath.Join(tmpDir, config.RequirementsConfigFileName), "merged file")
}

func TestRequirementsMergeConfigMap(t *testing.T) {
	// setup the disk
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	srcDir := filepath.Join("test_data")
	require.DirExists(t, srcDir)

	err = files.CopyDirOverwrite(srcDir, tmpDir)
	require.NoError(t, err, "failed to copy %s to %s", srcDir, tmpDir)

	changesFile := filepath.Join("test_data", "changes.yml")
	require.FileExists(t, changesFile)

	changesYaml, err := ioutil.ReadFile(changesFile)
	require.NoError(t, err, "failed to load %s", changesYaml)

	// now lets run the command
	_, o := merge.NewCmdRequirementsMerge()
	o.Dir = tmpDir
	o.KubeClient = fake.NewSimpleClientset(
		&corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      merge.ConfigMapName,
				Namespace: merge.ConfigMapNamespace,
			},
			Data: map[string]string{
				merge.ConfigMapKey: string(changesYaml),
			},
		},
	)

	t.Logf("merging requirements from ConfigMap in dir %s\n", tmpDir)

	err = o.Run()
	require.NoError(t, err, "failed to run merge")

	testhelpers.AssertTextFilesEqual(t, filepath.Join(tmpDir, "expected.yml"), filepath.Join(tmpDir, config.RequirementsConfigFileName), "merged file")
}

func TestRequirementsMergeConfigMapDoesNotExist(t *testing.T) {
	// setup the disk
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	srcDir := filepath.Join("test_data")
	require.DirExists(t, srcDir)

	err = files.CopyDirOverwrite(srcDir, tmpDir)
	require.NoError(t, err, "failed to copy %s to %s", srcDir, tmpDir)

	// now lets run the command
	_, o := merge.NewCmdRequirementsMerge()
	o.Dir = tmpDir
	o.KubeClient = fake.NewSimpleClientset()

	t.Logf("merging requirements from ConfigMap in dir %s\n", tmpDir)

	err = o.Run()
	require.NoError(t, err, "failed to run merge")
}
