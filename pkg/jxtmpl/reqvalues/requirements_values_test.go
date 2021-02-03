package reqvalues

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"

	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/stretchr/testify/assert"
)

func TestSaveRequirementsValuesFile(t *testing.T) {
	dir := t.TempDir()
	err := files.CopyDir("test_data", dir, true)
	assert.NoError(t, err)

	c := &jxcore.RequirementsConfig{}
	err = SaveRequirementsValuesFile(c, dir, filepath.Join(dir, "jx-global-values.yaml"))
	assert.NoError(t, err)

	testhelpers.AssertTextFilesEqual(t, filepath.Join(dir, "jx-global-values-expected.yaml"), filepath.Join(dir, "jx-global-values.yaml"), "jx-global-values are not as expected")
}
