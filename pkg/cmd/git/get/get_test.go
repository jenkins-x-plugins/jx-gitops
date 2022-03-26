package get_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/git/get"
	"github.com/jenkins-x/go-scm/scm/driver/fake"
	jxfake "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/stretchr/testify/require"
)

func TestGitGetFromRepository(t *testing.T) {
	tmpDir := t.TempDir()

	_, o := get.NewCmdGitGet()

	o.SourceURL = "https://github.com/jenkins-x-plugins/jx-gitops"
	o.Branch = "master"
	o.ScmClient, _ = fake.NewDefault()
	o.Dir = tmpDir
	o.Path = "expected.txt"
	o.FromRepository = "myorg/myrepo"

	err := o.Run()
	if err != nil {
		require.NoError(t, err, "failed to run command")
	}

	generated := filepath.Join(tmpDir, o.Path)
	require.FileExists(t, generated, "should have generated file")

	t.Logf("generated file %s\n", generated)
}

func TestGitGetFromEnvironment(t *testing.T) {
	tmpDir := t.TempDir()

	ns := "jx"
	devEnv := jxenv.CreateDefaultDevEnvironment(ns)
	devEnv.Namespace = ns
	devEnv.Spec.Source.URL = "https://github.com/myorg/myrepo.git"

	jxClient := jxfake.NewSimpleClientset(devEnv)

	_, o := get.NewCmdGitGet()

	o.SourceURL = "https://github.com/jenkins-x-plugins/jx-gitops"
	o.Branch = "master"
	o.ScmClient, _ = fake.NewDefault()
	o.JXClient = jxClient
	o.Dir = tmpDir
	o.Path = "expected.txt"
	o.Env = "dev"
	o.Namespace = ns

	err := o.Run()
	if err != nil {
		require.NoError(t, err, "failed to run command")
	}

	generated := filepath.Join(tmpDir, o.Path)
	require.FileExists(t, generated, "should have generated file")

	t.Logf("generated file %s\n", generated)
}
