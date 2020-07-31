package edit_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/extsecret/edit"
	"github.com/stretchr/testify/assert"

	"github.com/jenkins-x/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x/jx-gitops/pkg/secretmapping"
	"github.com/stretchr/testify/require"

	"github.com/jenkins-x/jx/v2/pkg/util"
)

func TestCmdSecretsMappingEdit(t *testing.T) {

	tests := []struct {
		name     string
		args     []string
		callback func(t *testing.T, sm *v1alpha1.SecretMapping)
		wantErr  bool
		fail     bool
	}{
		{
			name: "gsm_defaults_add",
			args: []string{"--gcp-project-id=foo", "--cluster-name=bar"},
			callback: func(t *testing.T, sm *v1alpha1.SecretMapping) {
				assert.Equal(t, 2, len(sm.Spec.Secrets), "should have found 2 mappings")
				for _, secret := range sm.Spec.Secrets {
					assert.Equal(t, "foo", secret.GcpSecretsManager.ProjectId, "secret.GcpSecretsManager.ProjectId")
					assert.Equal(t, "bar", secret.GcpSecretsManager.UniquePrefix, "secret.GcpSecretsManager.UniquePrefix")
					assert.Equal(t, "latest", secret.GcpSecretsManager.Version, "secret.GcpSecretsManager.Version")
				}
			},
		},
		{
			name: "gsm_defaults_dont_replace",
			args: []string{"--gcp-project-id=foo", "--cluster-name=bar"},
			callback: func(t *testing.T, sm *v1alpha1.SecretMapping) {
				assert.Equal(t, 2, len(sm.Spec.Secrets), "should have found 2 mappings")
				assert.Equal(t, "phill", sm.Spec.Secrets[0].GcpSecretsManager.ProjectId, "secret.GcpSecretsManager.ProjectId")
				assert.Equal(t, "collins", sm.Spec.Secrets[0].GcpSecretsManager.UniquePrefix, "secret.GcpSecretsManager.UniquePrefix")
				assert.Equal(t, "1", sm.Spec.Secrets[0].GcpSecretsManager.Version, "secret.GcpSecretsManager.Version")
				assert.Equal(t, "foo", sm.Spec.Secrets[1].GcpSecretsManager.ProjectId, "secret.GcpSecretsManager.ProjectId")
				assert.Equal(t, "latest", sm.Spec.Secrets[1].GcpSecretsManager.Version, "secret.GcpSecretsManager.Version")

			},
		},
	}
	tmpDir, err := ioutil.TempDir("", "jx-cmd-sec-")
	require.NoError(t, err, "failed to create temp dir")
	require.DirExists(t, tmpDir, "could not create temp dir for running tests")

	for i, tt := range tests {
		if tt.name == "" {
			tt.name = fmt.Sprintf("test%d", i)
		}
		t.Logf("running test %s", tt.name)
		dir := filepath.Join(tmpDir)

		err = os.MkdirAll(dir, util.DefaultWritePermissions)
		require.NoError(t, err, "failed to create dir %s", dir)

		localSecretsFile := filepath.Join("test_data", tt.name)
		err = util.CopyDir(localSecretsFile, dir, true)
		require.NoError(t, err, "failed to copy %s to %s", localSecretsFile, dir)

		cmd, _ := edit.NewCmdSecretMappingEdit()
		args := append(tt.args, "--dir", dir)

		err := cmd.ParseFlags(args)
		require.NoError(t, err, "failed to parse arguments %#v for test %s", args, tt.name)

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

		secretMapping, _, err := secretmapping.LoadSecretMapping(dir, true)
		require.NoError(t, err, "failed to load requirements from dir %s", dir)

		if tt.callback != nil {
			tt.callback(t, secretMapping)
		}
	}
}
