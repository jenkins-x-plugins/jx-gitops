package templater_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/apps/templater"
	"github.com/jenkins-x/jx-helpers/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

func TestTemplater(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "values.yaml-")
	require.NoError(t, err, "failed to create temp file")
	tmpFileName := tmpFile.Name()

	source := filepath.Join("test_data", "values.yaml.gotmpl")
	require.FileExists(t, source)

	requirements, _, err := config.LoadRequirementsConfig("test_data", true)
	require.NoError(t, err, "failed to load requirements")

	tmpl := templater.NewTemplater(requirements, []string{filepath.Join("test_data", "secrets.yaml")})

	err = tmpl.Generate(source, tmpFileName)
	require.NoError(t, err, "failed to generate template %s")

	t.Logf("generated %s to %s\n", source, tmpFileName)

	testhelpers.AssertTextFilesEqual(t, filepath.Join("test_data", "values.yaml"), tmpFileName, "generated file")
}
