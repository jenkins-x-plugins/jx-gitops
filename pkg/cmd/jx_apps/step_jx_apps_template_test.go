package jx_apps_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/jx_apps"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/cli"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepJxAppsTemplate(t *testing.T) {
	_, o := jx_apps.NewCmdJxAppsTemplate()

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "failed to create tmp dir")

	o.Dir = filepath.Join("test_data", "input")
	o.OutDir = tmpDir
	o.VersionStreamDir = filepath.Join("test_data", "versionstream")
	runner := &fakerunner.FakeRunner{
		CommandRunner: func(c *cmdrunner.Command) (string, error) {
			if c.Name == "clone" && len(c.Args) > 0 {
				// lets really git clone but then fake out all other commands
				return cmdrunner.DefaultCommandRunner(c)
			}
			return "", nil
		},
	}
	o.Gitter = cli.NewCLIClient("", runner.Run)

	err = o.Run()
	require.NoError(t, err, "failed to run the command")

	templateDir := tmpDir
	require.DirExists(t, templateDir)

	t.Logf("generated templates to %s", templateDir)

	assert.FileExists(t, filepath.Join(templateDir, "foo", "external-dns", "deployment.yaml"))
	assert.FileExists(t, filepath.Join(templateDir, "foo", "external-dns", "service.yaml"))
	assert.FileExists(t, filepath.Join(templateDir, "foo", "external-dns", "clusterrolebinding.yaml"))

}
