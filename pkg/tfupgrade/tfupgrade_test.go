package tfupgrade_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/tfupgrade"
	jxfake "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

func TestTerraformUpgrade(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	srcDir := "test_data"
	err = files.CopyDirOverwrite(srcDir, tmpDir)
	require.NoError(t, err, "failed to copy %s to %s", srcDir, tmpDir)

	o := &tfupgrade.Options{}
	o.JXClient = jxfake.NewSimpleClientset()
	o.Namespace = "ns"
	o.Dir = tmpDir

	err = o.Run()
	require.NoError(t, err, "failed to run in dir %s", srcDir, tmpDir)

	testhelpers.AssertEqualFileText(t, filepath.Join(tmpDir, "expected.tf"), filepath.Join(tmpDir, "main.tf"))
}
