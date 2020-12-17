package update_test

import (
	"path/filepath"
	"reflect"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/kpt/update"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/stretchr/testify/require"
)

func TestUpdateKptNoFilter(t *testing.T) {
	sourceDir := filepath.Join("test_data")
	absSourceDir, err := filepath.Abs(sourceDir)
	require.NoError(t, err, "failed to find abs dir of %s", sourceDir)
	require.DirExists(t, absSourceDir)

	_, uk := update.NewCmdKptUpdate()
	uk.KptBinary = "kpt"

	runner := &fakerunner.FakeRunner{}
	uk.CommandRunner = runner.Run
	uk.Dir = sourceDir
	uk.Version = "master"
	err = uk.Run()
	require.NoError(t, err, "failed to run update kpt")

	runner.ExpectResults(t,
		fakerunner.FakeResult{
			CLI: "git status -s",
		},
		fakerunner.FakeResult{
			CLI: "kpt pkg update config-root/namespaces/app1@master --strategy alpha-git-patch",
			Dir: absSourceDir,
		},
		fakerunner.FakeResult{
			CLI: "kpt pkg update config-root/namespaces/app2@master --strategy alpha-git-patch",
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
	uk.KptBinary = "kpt"

	runner := &fakerunner.FakeRunner{}
	uk.CommandRunner = runner.Run
	uk.Dir = sourceDir
	uk.RepositoryURL = "https://github.com/another/thing"
	uk.Version = "master"

	err = uk.Run()
	require.NoError(t, err, "failed to run update kpt")

	runner.ExpectResults(t,
		fakerunner.FakeResult{
			CLI: "git status -s",
		},
		fakerunner.FakeResult{
			CLI: "kpt pkg update config-root/namespaces/app2@master --strategy alpha-git-patch",
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
	uk.KptBinary = "kpt"

	runner := &fakerunner.FakeRunner{}
	uk.CommandRunner = runner.Run
	uk.Dir = sourceDir
	uk.RepositoryName = "jxr-kube-resources"
	uk.Version = "master"

	err = uk.Run()
	require.NoError(t, err, "failed to run update kpt")

	runner.ExpectResults(t,
		fakerunner.FakeResult{
			CLI: "git status -s",
		},
		fakerunner.FakeResult{
			CLI: "kpt pkg update config-root/namespaces/app1@master --strategy alpha-git-patch",
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
	uk.KptBinary = "kpt"

	runner := &fakerunner.FakeRunner{}
	uk.CommandRunner = runner.Run
	uk.Dir = sourceDir
	uk.RepositoryURL = "https://does/not/exist.git"
	uk.Version = "master"

	err = uk.Run()
	require.NoError(t, err, "failed to run update kpt")

	runner.ExpectResults(t,
		fakerunner.FakeResult{
			CLI: "git status -s",
		},
	)
}

func TestOptions_loadOverrideStrategies(t *testing.T) {

	tests := []struct {
		name    string
		want    map[string]string
		wantErr bool
	}{
		{name: "validate_pass", want: map[string]string{"foo": "bar", "cheese": "wine", "versionStream": "force-delete-replace"}, wantErr: false},
		{name: "validate_fail", want: map[string]string{"versionStream": "force-delete-replace"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			o := &update.Options{
				Dir: filepath.Join("test_data", tt.name),
			}
			got, err := o.LoadOverrideStrategies()
			if (err != nil) != tt.wantErr {
				t.Errorf("loadOverrideStrategies() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loadOverrideStrategies() got = %v, want %v", got, tt.want)
			}
		})
	}
}
