package tfupgrade_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/tfupgrade"
	jxfake "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTerraformUpgrade(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	fileNames, err := ioutil.ReadDir("test_data")
	assert.NoError(t, err)

	for _, f := range fileNames {
		if !f.IsDir() {
			continue
		}
		name := f.Name()
		srcDir := filepath.Join("test_data", name)
		require.DirExists(t, srcDir)

		err = files.CopyDirOverwrite(srcDir, tmpDir)
		require.NoError(t, err, "failed to copy %s to %s", srcDir, tmpDir)

		o := &tfupgrade.Options{}
		o.JXClient = jxfake.NewSimpleClientset()
		o.Namespace = "ns"
		o.Dir = tmpDir

		err = o.Run()
		require.NoError(t, err, "failed to run in dir %s for %s", srcDir, name)

		testhelpers.AssertEqualFileText(t, filepath.Join(tmpDir, "expected.tf"), filepath.Join(tmpDir, "main.tf"))
	}
}

func TestTerraformUpgradeReplaceValue(t *testing.T) {
	testCases := []struct {
		input    string
		expected string
	}{
		{
			input:    "not-even-url",
			expected: "",
		},
		{
			input:    "github.com/jenkins-x/terraform-google-jx",
			expected: "github.com/jenkins-x/terraform-google-jx?ref=v1.9.0",
		},
		{
			input:    "github.com/jenkins-x/terraform-google-jx",
			expected: "github.com/jenkins-x/terraform-google-jx?ref=v1.9.0",
		},
		{
			input:    "github.com/jenkins-x/terraform-google-jx?ref=master",
			expected: "github.com/jenkins-x/terraform-google-jx?ref=v1.9.0",
		},
		{
			input:    "https://github.com/jenkins-x/terraform-google-jx?ref=master",
			expected: "https://github.com/jenkins-x/terraform-google-jx?ref=v1.9.0",
		},
		{
			input:    "git::https://github.com/jenkins-x/terraform-google-jx?ref=master",
			expected: "git::https://github.com/jenkins-x/terraform-google-jx?ref=v1.9.0",
		},
	}
	for _, tc := range testCases {
		o := &tfupgrade.Options{
			VersionStreamDir: filepath.Join("test_data", "gke", "versionStream"),
		}

		got := o.ReplaceValue(tc.input)
		assert.Equal(t, tc.expected, got, "for git URL %s", tc.input)
	}
}
