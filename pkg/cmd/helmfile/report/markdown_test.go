package report_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helmfile/report"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/releasereport"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/stretchr/testify/require"
)

// generateTestOutput enable to regenerate the expected output
var generateTestOutput = false

func TestHemlfileMarkdownReport(t *testing.T) {
	var charts []*releasereport.NamespaceReleases

	tmpDir := t.TempDir()

	sourceFile := filepath.Join("testdata", "releases.yaml")
	expectedPath := filepath.Join("testdata", "expected.README.md")

	err := yamls.LoadFile(sourceFile, &charts)
	require.NoError(t, err, "failed to load file %s", sourceFile)
	require.NotEmpty(t, charts, "no namespace charts found for %s", sourceFile)

	md, err := report.ToMarkdown(charts)
	require.NoError(t, err, "failed to generate markdown for file %s", sourceFile)

	generatedFile := filepath.Join(tmpDir, "README.md")
	err = os.WriteFile(generatedFile, []byte(md), files.DefaultFileWritePermissions)
	require.NoError(t, err, "failed to save file %s", generatedFile)

	if generateTestOutput {
		data, err := os.ReadFile(generatedFile)
		require.NoError(t, err, "failed to load %s", generatedFile)

		err = os.WriteFile(expectedPath, data, 0o600)
		require.NoError(t, err, "failed to save file %s", expectedPath)

		t.Logf("saved file %s\n", expectedPath)
		return
	}

	testhelpers.AssertTextFilesEqual(t, expectedPath, generatedFile, "generated README.md")
}
