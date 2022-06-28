package validate_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helmfile/validate"
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

		tmpDir := t.TempDir()

		srcDir := filepath.Join("testdata", tc.testFolder)
		require.DirExists(t, srcDir)

		err := files.CopyDirOverwrite(srcDir, tmpDir)
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
