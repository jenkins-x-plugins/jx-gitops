package namespace_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/namespace"
	"github.com/jenkins-x/jx-gitops/pkg/kyamls"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

func TestUpdateNamespaceInYamlFiles(t *testing.T) {
	sourceData := filepath.Join("test_data", "regular")
	files, err := ioutil.ReadDir(sourceData)
	assert.NoError(t, err)

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	type testCase struct {
		SourceFile   string
		ResultFile   string
		ExpectedFile string
	}

	var testCases []testCase
	for _, f := range files {
		if f.IsDir() {
			name := f.Name()
			srcFile := filepath.Join(sourceData, name, "source.yaml")
			expectedFile := filepath.Join(sourceData, name, "expected.yaml")
			require.FileExists(t, srcFile)
			require.FileExists(t, expectedFile)

			outFile := filepath.Join(tmpDir, name+".yaml")
			err = util.CopyFile(srcFile, outFile)
			require.NoError(t, err, "failed to copy %s to %s", srcFile, outFile)

			testCases = append(testCases, testCase{
				SourceFile:   srcFile,
				ResultFile:   outFile,
				ExpectedFile: expectedFile,
			})
		}
	}

	err = namespace.UpdateNamespaceInYamlFiles(tmpDir, "something", kyamls.Filter{})
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

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	err = util.CopyDirOverwrite(srcFile, tmpDir)
	require.NoError(t, err, "failed to copy %s to %s", srcFile, tmpDir)

	o := &namespace.Options{
		Dir:     tmpDir,
		DirMode: true,
	}

	err = o.Run()
	require.NoError(t, err, "failed to run in dir %s", srcFile, tmpDir)

	t.Logf("replaced namespaces in dir %s\n", tmpDir)

	found := map[string][]string{}
	err = filepath.Walk(tmpDir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		relPath, err := filepath.Rel(tmpDir, path)
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

	for k, v := range found {
		t.Logf("found files for namespace %s = %#v", k, v)
		assert.Len(t, v, 1, "files in namespace %s", k)
	}
	assert.Len(t, found, 2, "found namespaces")
}
