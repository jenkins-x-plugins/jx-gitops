//nolint:dupl
package update_test

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/kpt/update"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestUpdateKptNoFilter(t *testing.T) {
	sourceDir := "testdata"
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
	sourceDir := "testdata"
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
			CLI: "kpt pkg update config-root/namespaces/app2@master --strategy resource-merge",
			Dir: absSourceDir,
		},
	)
}

func TestUpdateKptFilterRepositoryName(t *testing.T) {
	sourceDir := "testdata"
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
			CLI: "kpt pkg update config-root/namespaces/app1@master --strategy resource-merge",
			Dir: absSourceDir,
		},
	)
}

func TestUpdateKptFilterNotMatching(t *testing.T) {
	sourceDir := "testdata"
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

// failOnFnEval returns a yaml content error for the `kpt fn eval` command that migrates
// old-format Kptfiles, mimicking a package containing YAML that kpt cannot parse (e.g. Helm templates)
func failOnFnEval(c *cmdrunner.Command) (string, error) {
	if strings.Contains(cmdrunner.CLI(c), "fn eval") {
		return "", errors.New("MalformedYAMLError: yaml: did not find expected node content")
	}
	return "", nil
}

func TestUpdateKptUpgradesLegacyKptfileWithPinnedImage(t *testing.T) {
	sourceDir := "testdata"

	_, uk := update.NewCmdKptUpdate()
	uk.KptBinary = "kpt"

	runner := &fakerunner.FakeRunner{}
	uk.CommandRunner = runner.Run
	uk.Dir = sourceDir
	// leaving Version empty triggers the Kptfile format upgrade via `kpt fn eval`

	err := uk.Run()
	require.NoError(t, err, "failed to run update kpt")

	// the testdata Kptfiles use apiVersion kpt.dev/v1alpha1 so each is migrated via `kpt fn eval`
	var fnEvalCommands []string
	for _, c := range runner.OrderedCommands {
		cli := cmdrunner.CLI(c)
		if strings.Contains(cli, "fn eval") {
			fnEvalCommands = append(fnEvalCommands, cli)
		}
	}
	require.NotEmpty(t, fnEvalCommands, "expected at least one `kpt fn eval` command to migrate legacy Kptfiles")
	for _, cli := range fnEvalCommands {
		require.Contains(t, cli, "--image ghcr.io/kptdev/krm-functions-catalog/fix:ad0c3fe",
			"fn eval must use the pinned krm-functions-catalog fix image")
	}
}

func TestUpdateKptIgnoreYamlErrorSkipsPackage(t *testing.T) {
	sourceDir := "testdata"

	_, uk := update.NewCmdKptUpdate()
	uk.KptBinary = "kpt"

	runner := &fakerunner.FakeRunner{CommandRunner: failOnFnEval}
	uk.CommandRunner = runner.Run
	uk.Dir = sourceDir
	// leaving Version empty triggers the Kptfile format upgrade via `kpt fn eval`
	uk.IgnoreYamlContentError = true

	err := uk.Run()
	require.NoError(t, err, "should ignore the yaml content error when --ignore-yaml-error is set")
}

func TestUpdateKptYamlErrorWithoutFlagFails(t *testing.T) {
	sourceDir := "testdata"

	_, uk := update.NewCmdKptUpdate()
	uk.KptBinary = "kpt"

	runner := &fakerunner.FakeRunner{CommandRunner: failOnFnEval}
	uk.CommandRunner = runner.Run
	uk.Dir = sourceDir
	// leaving Version empty triggers the Kptfile format upgrade via `kpt fn eval`

	err := uk.Run()
	require.Error(t, err, "should surface the yaml content error when --ignore-yaml-error is not set")
	require.Contains(t, err.Error(), "yaml: did not find expected node content")
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
				Dir: filepath.Join("testdata", tt.name),
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
