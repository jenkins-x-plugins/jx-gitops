package namespace_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/namespace"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestUpdateNamespaceInYamlFiles(t *testing.T) {
	sourceData := filepath.Join("test_data", "regular")
	fileNames, err := ioutil.ReadDir(sourceData)
	assert.NoError(t, err)

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	type testCase struct {
		SourceFile   string
		ResultFile   string
		ExpectedFile string
	}

	var testCases []testCase
	for _, f := range fileNames {
		if !f.IsDir() {
			continue
		}
		name := f.Name()
		srcFile := filepath.Join(sourceData, name, "source.yaml")
		expectedFile := filepath.Join(sourceData, name, "expected.yaml")
		require.FileExists(t, srcFile)
		require.FileExists(t, expectedFile)

		outFile := filepath.Join(tmpDir, name+".yaml")
		err = files.CopyFile(srcFile, outFile)
		require.NoError(t, err, "failed to copy %s to %s", srcFile, outFile)

		testCases = append(testCases, testCase{
			SourceFile:   srcFile,
			ResultFile:   outFile,
			ExpectedFile: expectedFile,
		})

	}

	_, err = namespace.UpdateNamespaceInYamlFiles(tmpDir, tmpDir, "something", &kyamls.Filter{}, false)
	require.NoError(t, err, "failed to update namespace in dir %s", tmpDir)

	for _, tc := range testCases {
		resultData, err := ioutil.ReadFile(tc.ResultFile)
		require.NoError(t, err, "failed to load results %s", tc.ResultFile)

		expectData, err := ioutil.ReadFile(tc.ExpectedFile)
		require.NoError(t, err, "failed to load results %s", tc.ExpectedFile)

		result := strings.TrimSpace(string(resultData))
		expectedText := strings.TrimSpace(string(expectData))
		if d := cmp.Diff(result, expectedText); d != "" {
			t.Errorf("Generated Pipeline did not match expected: %s", d)
		}
		t.Logf("generated for file %s file %s\n", tc.SourceFile, result)
	}
}

func TestNamespaceDirMode(t *testing.T) {
	srcFile := filepath.Join("test_data", "dirmode")
	require.DirExists(t, srcFile)

	rootTmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	tmpDir := filepath.Join(rootTmpDir, "namespaces")
	err = os.MkdirAll(tmpDir, files.DefaultDirWritePermissions)
	require.NoError(t, err, "failed to make namespaces dir")

	err = files.CopyDirOverwrite(srcFile, tmpDir)
	require.NoError(t, err, "failed to copy %s to %s", srcFile, tmpDir)

	o := &namespace.Options{
		Dir:     tmpDir,
		DirMode: true,
	}

	err = o.Run()
	require.NoError(t, err, "failed to run in dir %s", srcFile, tmpDir)

	t.Logf("replaced namespaces in dir %s\n", tmpDir)

	found := map[string][]string{}
	err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error { //nolint:staticcheck
		if info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		relPath, err := filepath.Rel(tmpDir, path) //nolint:staticcheck
		if err != nil {
			return errors.Wrapf(err, "failed to find relative path of %s", path)
		}
		paths := strings.Split(relPath, string(os.PathSeparator))
		ns := paths[0]

		node, err := yaml.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to load file %s", relPath)
		}

		nsNode, err := node.Pipe(yaml.Lookup("metadata", "namespace"))
		if err != nil {
			return errors.Wrapf(err, "failed to find namespace at path %s", relPath)
		}

		if nsNode != nil {
			nsNodeText, err := nsNode.String()
			if err != nil {
				return errors.Wrapf(err, "failed to find namespace text at %s", relPath)
			}
			actualNS := strings.TrimSpace(nsNodeText)
			if assert.Equal(t, ns, actualNS, "namespace of %s", relPath) {
				found[ns] = append(found[ns], relPath)
			}
		}
		return nil
	})
	require.NoError(t, err, "failed to find results")

	clusterNamespacesDir := filepath.Join(rootTmpDir, "cluster", "namespaces")
	assert.DirExists(t, clusterNamespacesDir, "should have created folder for the lazy created Namespace resources")

	for k, v := range found {
		t.Logf("found files for namespace %s = %#v", k, v)
		assert.Len(t, v, 1, "files in namespace %s", k)

		// lets assert we have a namespace file
		nsFile := filepath.Join(clusterNamespacesDir, k+".yaml")
		if assert.FileExists(t, nsFile) {
			t.Logf("lazy created the namespace file %s", nsFile)
		}
	}
	assert.Len(t, found, 2, "found namespaces")
}

func TestShouldPreserveNamespace(t *testing.T) {
	path := "test_data/dirmode-with-exceptions/jx/service-preserve.yaml"
	rNode, readErr := yaml.ReadFile(path)
	result := namespace.ShouldPreserveNamespace(rNode, path)

	assert.Nil(t, readErr)
	assert.True(t, result)
}

func TestShouldPreserveNamespace_WithoutAnnotationWillNotKeepOriginalNamespace(t *testing.T) {
	path := "test_data/dirmode-with-exceptions/jx/service-do-not-preserve.yaml"
	rNode, readErr := yaml.ReadFile(path)
	result := namespace.ShouldPreserveNamespace(rNode, path)

	assert.Nil(t, readErr)
	assert.False(t, result)
}

func TestGetNamespaceToPreserveIfShouldKeepIt(t *testing.T) {
	path := "test_data/dirmode-with-exceptions/jx/service-preserve.yaml"
	rNode, readErr := yaml.ReadFile(path)
	ns := namespace.GetNamespaceToPreserveIfShouldKeepIt(rNode, path)

	assert.Nil(t, readErr)
	assert.Equal(t, "some-other-namespace", ns)
}

func TestMoveToTargetNamespace(t *testing.T) {
	mock := osToolsMock{}

	_ = namespace.MoveToTargetNamespace(
		"test_data/dirmode-with-exceptions",
		"test_data/dirmode-with-exceptions/jx/service-preserve.yaml",
		"some-other-namespace",
		"jx",
		&mock)

	assert.Contains(t, mock.calls[0], "test_data/dirmode-with-exceptions/some-other-namespace && chmod -rwxr-xr-x")
	assert.Contains(t, mock.calls[1], "test_data/dirmode-with-exceptions/some-other-namespace/service-preserve.yaml")
}

type osToolsMock struct {
	calls []string
}

func (o *osToolsMock) MkdirAll(path string, perm os.FileMode) error {
	o.calls = append(o.calls, fmt.Sprintf("mkdir -p %s && chmod %s", path, perm.String()))
	return nil
}

func (o *osToolsMock) Rename(oldpath, newpath string) error {
	o.calls = append(o.calls, fmt.Sprintf("mv %s %s", oldpath, newpath))
	return nil
}
