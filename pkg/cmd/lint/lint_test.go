package lint_test

import (
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/lint"
	"github.com/stretchr/testify/require"
)

func TestLint(t *testing.T) {
	_, o := lint.NewCmdLint()
	o.Dir = "test_data"

	err := o.Run()
	require.NoError(t, err, "failed to run")
}
