package templater_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"

	"github.com/jenkins-x/jx-gitops/pkg/jxtmpl/templater"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

func TestTemplater(t *testing.T) {
	tmpFile, err := ioutil.TempFile("", "values.yaml-")
	require.NoError(t, err, "failed to create temp file")
	tmpFileName := tmpFile.Name()

	source := filepath.Join("test_data", "values.yaml.gotmpl")
	require.FileExists(t, source)

	requirementsResource, _, err := jxcore.LoadRequirementsConfig("test_data", true)
	require.NoError(t, err, "failed to load requirements")
	requirements := &requirementsResource.Spec
	tmpl := templater.NewTemplater(requirements, []string{filepath.Join("test_data", "secrets.yaml")})

	err = tmpl.Generate(source, tmpFileName)
	require.NoError(t, err, "failed to generate template %s")

	t.Logf("generated %s to %s\n", source, tmpFileName)

	testhelpers.AssertTextFilesEqual(t, filepath.Join("test_data", "values.yaml"), tmpFileName, "generated file")
}
