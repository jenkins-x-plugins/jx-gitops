package merge_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/requirement/merge"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

var (
	// generateTestOutput enable to regenerate the expected output
	generateTestOutput = false
)

func TestRequirementsMergeFile(t *testing.T) {
	// setup the disk
	rootTmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	fileNames, err := ioutil.ReadDir("test_data")
	assert.NoError(t, err)

	for _, f := range fileNames {
		if f.IsDir() {
			name := f.Name()
			srcDir := filepath.Join("test_data", name)
			require.DirExists(t, srcDir)

			tmpDir := filepath.Join(rootTmpDir, name)
			err = files.CopyDirOverwrite(srcDir, tmpDir)
			require.NoError(t, err, "failed to copy %s to %s", srcDir, tmpDir)

			// now lets run the command
			_, o := merge.NewCmdRequirementsMerge()
			o.Dir = tmpDir
			o.File = filepath.Join(tmpDir, "changes.yml")

			t.Logf("merging requirements in dir %s\n", tmpDir)

			err = o.Run()
			require.NoError(t, err, "failed to run merge")

			expectedPath := filepath.Join(srcDir, "expected.yml")
			generatedFile := filepath.Join(tmpDir, jxcore.RequirementsConfigFileName)

			if generateTestOutput {
				data, err := ioutil.ReadFile(generatedFile)
				require.NoError(t, err, "failed to load %s", generatedFile)

				err = ioutil.WriteFile(expectedPath, data, 0666)
				require.NoError(t, err, "failed to save file %s", expectedPath)
				continue
			}
			testhelpers.AssertTextFilesEqual(t, expectedPath, generatedFile, "merged file for test "+name)
		}
	}
}

func TestRequirementsMergeConfigMap(t *testing.T) {
	// setup the disk
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	srcDir := filepath.Join("test_data", "sample")
	require.DirExists(t, srcDir)

	err = files.CopyDirOverwrite(srcDir, tmpDir)
	require.NoError(t, err, "failed to copy %s to %s", srcDir, tmpDir)

	changesFile := filepath.Join("test_data", "sample", "changes.yml")
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

	testhelpers.AssertTextFilesEqual(t, filepath.Join(tmpDir, "expected.yml"), filepath.Join(tmpDir, jxcore.RequirementsConfigFileName), "merged file")
}

func TestRequirementsMergeConfigMapDoesNotExist(t *testing.T) {
	// setup the disk
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	srcDir := filepath.Join("test_data", "sample")
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
