package build_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/helm/build"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepHelmBuild(t *testing.T) {
	sourceData := filepath.Join("test_data")
	fileNames, err := ioutil.ReadDir(sourceData)
	assert.NoError(t, err)

	for _, f := range fileNames {
		if f.IsDir() {
			name := f.Name()
			path := filepath.Join(sourceData, name)

			t.Logf("running test dir %s", name)

			runner := &fakerunner.FakeRunner{}
			helmBin := "helm"

			_, o := build.NewCmdHelmBuild()
			o.HelmBinary = helmBin
			o.CommandRunner = runner.Run
			o.ChartsDir = filepath.Join(path, "charts")

			err = o.Run()
			require.NoError(t, err, "failed to run the command for dir %s", name)

			for _, c := range runner.OrderedCommands {
				t.Logf("ran: %s\n", c.CLI())
			}
		}
	}
}
