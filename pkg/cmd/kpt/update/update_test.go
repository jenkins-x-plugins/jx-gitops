package update_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/kpt/update"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner/fakerunner"
	"github.com/stretchr/testify/require"
)

func TestUpdateKptNoFilter(t *testing.T) {
	sourceDir := filepath.Join("test_data")
	absSourceDir, err := filepath.Abs(sourceDir)
	require.NoError(t, err, "failed to find abs dir of %s", sourceDir)
	require.DirExists(t, absSourceDir)

	_, uk := update.NewCmdKptUpdate()

	runner := &fakerunner.FakeRunner{}
	uk.CommandRunner = runner.Run
	uk.Dir = sourceDir

	err = uk.Run()
	require.NoError(t, err, "failed to run update kpt")

	runner.ExpectResults(t,
		fakerunner.FakeResult{
			CLI: "kpt pkg update config-root/namespaces/app1@master --strategy resource-merge",
			Dir: absSourceDir,
		},
		fakerunner.FakeResult{
			CLI: "kpt pkg update config-root/namespaces/app2@master --strategy resource-merge",
			Dir: absSourceDir,
		},
	)
}

func TestUpdateKptFilterRepositoryURL(t *testing.T) {
	sourceDir := filepath.Join("test_data")
	absSourceDir, err := filepath.Abs(sourceDir)
	require.NoError(t, err, "failed to find abs dir of %s", sourceDir)
	require.DirExists(t, absSourceDir)

	_, uk := update.NewCmdKptUpdate()

	runner := &fakerunner.FakeRunner{}
	uk.CommandRunner = runner.Run
	uk.Dir = sourceDir
	uk.RepositoryURL = "https://github.com/another/thing"

	err = uk.Run()
	require.NoError(t, err, "failed to run update kpt")

	runner.ExpectResults(t,
		fakerunner.FakeResult{
			CLI: "kpt pkg update config-root/namespaces/app2@master --strategy resource-merge",
			Dir: absSourceDir,
		},
	)
}
func TestUpdateKptFilterRepositoryName(t *testing.T) {
	sourceDir := filepath.Join("test_data")
	absSourceDir, err := filepath.Abs(sourceDir)
	require.NoError(t, err, "failed to find abs dir of %s", sourceDir)
	require.DirExists(t, absSourceDir)

	_, uk := update.NewCmdKptUpdate()

	runner := &fakerunner.FakeRunner{}
	uk.CommandRunner = runner.Run
	uk.Dir = sourceDir
	uk.RepositoryName = "jxr-kube-resources"

	err = uk.Run()
	require.NoError(t, err, "failed to run update kpt")

	runner.ExpectResults(t,
		fakerunner.FakeResult{
			CLI: "kpt pkg update config-root/namespaces/app1@master --strategy resource-merge",
			Dir: absSourceDir,
		},
	)
}

func TestUpdateKptFilterNotMatching(t *testing.T) {
	sourceDir := filepath.Join("test_data")
	absSourceDir, err := filepath.Abs(sourceDir)
	require.NoError(t, err, "failed to find abs dir of %s", sourceDir)
	require.DirExists(t, absSourceDir)

	_, uk := update.NewCmdKptUpdate()

	runner := &fakerunner.FakeRunner{}
	uk.CommandRunner = runner.Run
	uk.Dir = sourceDir
	uk.RepositoryURL = "https://does/not/exist.git"

	err = uk.Run()
	require.NoError(t, err, "failed to run update kpt")

	runner.ExpectResults(t)
}
