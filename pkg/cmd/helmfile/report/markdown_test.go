package report_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/report"
	"github.com/jenkins-x/jx-gitops/pkg/releasereport"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/stretchr/testify/require"
)

var (
	// generateTestOutput enable to regenerate the expected output
	generateTestOutput = false
)

func TestHemlfileMarkdownReport(t *testing.T) {
	var charts []*releasereport.NamespaceReleases

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "failed to create temp dir")

	sourceFile := filepath.Join("test_data", "releases.yaml")
	expectedPath := filepath.Join("test_data", "expected.README.md")

	err = yamls.LoadFile(sourceFile, &charts)
	require.NoError(t, err, "failed to load file %s", sourceFile)
	require.NotEmpty(t, charts, "no namespace charts found for %s", sourceFile)

	md, err := report.ToMarkdown(charts)
	require.NoError(t, err, "failed to generate markdown for file %s", sourceFile)

	generatedFile := filepath.Join(tmpDir, "README.md")
	err = ioutil.WriteFile(generatedFile, []byte(md), files.DefaultFileWritePermissions)
	require.NoError(t, err, "failed to save file %s", generatedFile)

	if generateTestOutput {
		data, err := ioutil.ReadFile(generatedFile)
		require.NoError(t, err, "failed to load %s", generatedFile)

		err = ioutil.WriteFile(expectedPath, data, 0666)
		require.NoError(t, err, "failed to save file %s", expectedPath)

		t.Logf("saved file %s\n", expectedPath)
		return
	}

	testhelpers.AssertTextFilesEqual(t, expectedPath, generatedFile, "generated README.md")
}
