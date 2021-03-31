package mirror_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helm/mirror"
	"github.com/stretchr/testify/require"
)

func TestHelmMirror(t *testing.T) {
	_, o := mirror.NewCmdMirror()
	o.Dir = filepath.Join("test_data", "versionStream")

	o.GitURL = "https://github.com/jenkins-x-test-projects/test-vertx-app"
	o.GitUsername = "jenkins-x-bot"
	o.GitToken = "dummy-token"
	o.NoPush = true
	err := o.Run()
	require.NoError(t, err, "failed to run")
}
