package validate_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/validate"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/stretchr/testify/require"
)

func TestStepHelmfileStructure(t *testing.T) {
	testCases := []struct {
		testFolder  string
		returnError bool
		errorString string
	}{
		{
			testFolder:  "input",
			returnError: false,
			errorString: "",
		},
		{
			testFolder:  "over_nested",
			returnError: true,
			errorString: "",
		},
		{
			testFolder:  "absolute_path",
			returnError: true,
			errorString: "",
		},
		{
			testFolder:  "wrong_namespace",
			returnError: true,
			errorString: "",
		},
		{
			testFolder:  "missing",
			returnError: true,
			errorString: "",
		},
		{
			testFolder:  "missing_repo",
			returnError: true,
			errorString: "",
		},
	}

	for _, tc := range testCases {

		tmpDir, err := ioutil.TempDir("", "")
		require.NoError(t, err, "failed to create tmp dir")

		srcDir := filepath.Join("test_data", tc.testFolder)
		require.DirExists(t, srcDir)

		err = files.CopyDirOverwrite(srcDir, tmpDir)
		require.NoError(t, err, "failed to copy generated helmfiles at %s to %s", srcDir, tmpDir)

		_, o := validate.NewCmdHelmfileValidate()
		o.Dir = tmpDir

		err = o.Run()
		if !tc.returnError {
			require.NoError(t, err)
		} else {
			require.Errorf(t, err, "error expected")
		}
	}

}
