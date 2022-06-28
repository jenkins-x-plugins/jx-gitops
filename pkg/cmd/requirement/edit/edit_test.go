//go:build unit
// +build unit

package edit_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/requirement/edit"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
)

func TestCmdRequirementsEdit(t *testing.T) {
	t.Parallel()

	type testData struct {
		name        string
		args        []string
		callback    func(t *testing.T, req *jxcore.RequirementsConfig)
		fail        bool
		initialFile string
	}

	gitOpsEnabled := filepath.Join("testdata", "gitops-enabled.yml")
	tests := []testData{
		{
			name: "bbs",
			args: []string{"--git-kind=bitbucketserver"},
			callback: func(t *testing.T, req *jxcore.RequirementsConfig) {
				assert.Equal(t, "bitbucketserver", req.Cluster.GitKind, "req.Cluster.GitKind")
			},
			initialFile: gitOpsEnabled,
		},
		{
			name: "bucket-logs",
			args: []string{"--bucket-logs", "gs://foo"},
			callback: func(t *testing.T, req *jxcore.RequirementsConfig) {
				assert.Equal(t, "gs://foo", req.GetStorageURL("logs"), "req.Storage.Logs.URL")
			},
			initialFile: gitOpsEnabled,
		},
		{
			name:        "bad-git-kind",
			args:        []string{"--git-kind=gitlob"},
			fail:        true,
			initialFile: gitOpsEnabled,
		},
		{
			name:        "bad-secret",
			args:        []string{"--secret=vaulx"},
			fail:        true,
			initialFile: gitOpsEnabled,
		},
	}

	tmpDir := t.TempDir()

	for i, tt := range tests {
		if tt.name == "" {
			tt.name = fmt.Sprintf("test%d", i)
		}
		t.Logf("running test %s", tt.name)
		dir := filepath.Join(tmpDir, tt.name)

		err := os.MkdirAll(dir, files.DefaultDirWritePermissions)
		require.NoError(t, err, "failed to create dir %s", dir)

		localReqFile := filepath.Join(dir, jxcore.RequirementsConfigFileName)
		if tt.initialFile != "" {
			err = files.CopyFile(tt.initialFile, localReqFile)
			require.NoError(t, err, "failed to copy %s to %s", tt.initialFile, localReqFile)
			require.FileExists(t, localReqFile, "file should have been copied")
		}

		cmd, _ := edit.NewCmdRequirementsEdit()
		args := append(tt.args, "--dir", dir)

		err = cmd.ParseFlags(args)
		require.NoError(t, err, "failed to parse arguments %#v for test %", args, tt.name)

		old := os.Args
		os.Args = args
		err = cmd.RunE(cmd, args)
		if err != nil {
			if tt.fail {
				t.Logf("got exected failure for test %s: %s", tt.name, err.Error())
				continue
			}
			t.Errorf("test %s reported error: %s", tt.name, err)
			continue
		}
		os.Args = old

		// now lets parse the requirements
		file := localReqFile
		require.FileExists(t, file, "should have generated the requirements file")

		req, _, err := jxcore.LoadRequirementsConfig(dir, jxcore.DefaultFailOnValidationError)
		require.NoError(t, err, "failed to load requirements from dir %s", dir)

		if tt.callback != nil {
			tt.callback(t, &req.Spec)
		}

	}

}
