package secret_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/sa/secret"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/yamls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestServiceAccountSecret(t *testing.T) {
	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "could not create temp dir")

	sourceData := filepath.Join("test_data")

	err = files.CopyDirOverwrite(sourceData, tmpDir)
	require.NoError(t, err, "failed to copy generated crds at %s to %s", sourceData, tmpDir)

	_, o := secret.NewCmdServiceAccountSecrets()

	o.File = filepath.Join(tmpDir, "sa.yaml")
	o.Secrets = []string{"cardiff", "mells"}

	err = o.Run()
	require.NoError(t, err, "failed to run")

	t.Logf("modified file %s\n", o.File)

	sa := &corev1.ServiceAccount{}
	err = yamls.LoadFile(o.File, sa)
	require.NoError(t, err, "failed load ServiceAccount %s", o.File)

	secrets := sa.Secrets

	assert.NotEmpty(t, secrets, "should have populated the file with secrets %s", o.File)
	for _, s := range secrets {
		t.Logf("has secret %s\n", s.Name)
	}
}
