package status_test

import (
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helmfile/status"
	"github.com/stretchr/testify/require"
)

func TestHemlfileStatus(t *testing.T) {
	_, o := status.NewCmdHelmfileStatus()
	o.Dir = "test_data"
	o.TestGitToken = "faketoken"
	err := o.Run()
	require.NoError(t, err, "failed to run")
}
