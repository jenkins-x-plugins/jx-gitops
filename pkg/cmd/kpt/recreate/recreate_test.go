package recreate_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/kpt/recreate"
	"github.com/jenkins-x/jx-gitops/pkg/testhelpers"
	"github.com/stretchr/testify/require"
)

func TestKptRecreate(t *testing.T) {
	sourceDir := filepath.Join("test_data")
	absSourceDir, err := filepath.Abs(sourceDir)
	require.NoError(t, err, "failed to find abs dir of %s", sourceDir)
	require.DirExists(t, absSourceDir)

	_, uk := recreate.NewCmdKptRecreate()

	runner := &testhelpers.FakeRunner{}
	uk.CommandRunner = runner.Run
	uk.Dir = sourceDir

	err = uk.Run()
	require.NoError(t, err, "failed to run recreate kpt")

	runner.ExpectResults(t,
		testhelpers.FakeResult{
			CLI: "kpt pkg get https://github.com/jenkins-x/jxr-kube-resources.git/jenkins-x/lighthouse@4cc6b80d49808060b1f06f530399b986ed344f23 config-root/namespaces/myapps/app1",
		},
		testhelpers.FakeResult{
			CLI: "kpt pkg get https://github.com/another/thing.git/kubernetes/app2@4cc6b80d49808060b1f06f530399b986ed344f23 config-root/namespaces/app2",
		},
	)
}
