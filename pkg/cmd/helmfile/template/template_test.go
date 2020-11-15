package template_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/template"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStepHelmfileTemplateFailsIfNoHelmfile(t *testing.T) {
	_, o := template.NewCmdHelmfileTemplate()
	o.Helmfile = filepath.Join("test_data", "does-not-exist.yaml")

	err := o.Run()
	require.Error(t, err, "should have failed due to missing helmfile")
}

func TestStepHelmfileTemplate(t *testing.T) {
	skipTestIfCommandFails(t, "helm", "version")
	skipTestIfCommandFails(t, "helmfile", "version")

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "failed to create tmp dir")

	srcDir := filepath.Join("test_data")
	require.DirExists(t, srcDir)

	err = files.CopyDirOverwrite(srcDir, tmpDir)
	require.NoError(t, err, "failed to copy generated crds at %s to %s", srcDir, tmpDir)

	_, o := template.NewCmdHelmfileTemplate()
	o.Dir = tmpDir
	o.Args = "--include-crds"

	err = o.Run()
	require.NoError(t, err, "failed to run the command")

	t.Logf("generated files to %s\n", o.OutputDir)

	assert.FileExists(t, filepath.Join(o.OutputDir, "namespaces", "jx", "tekton", "tekton-pipelines-controller-deploy.yaml"), "expected generated file")
	assert.FileExists(t, filepath.Join(o.OutputDir, "namespaces", "secret-infra", "kubernetes-external-secrets", "es-kubernetes-external-secrets-deploy.yaml"), "expected generated file")
	assert.FileExists(t, filepath.Join(o.OutputDir, "namespaces", "jx-production", "kubernetes-external-secrets", "es-kubernetes-external-secrets-deploy.yaml"), "expected generated file")

	assert.FileExists(t, filepath.Join(o.OutputDir, "cluster", "namespaces", "jx.yaml"), "expected generated namespace")
	assert.FileExists(t, filepath.Join(o.OutputDir, "cluster", "namespaces", "jx-production.yaml"), "expected generated namespace")
	assert.FileExists(t, filepath.Join(o.OutputDir, "cluster", "namespaces", "secret-infra.yaml"), "expected generated namespace")

	assert.FileExists(t, filepath.Join(o.OutputDir, "customresourcedefinitions", "secret-infra", "kubernetes-external-secrets", "externalsecrets.kubernetes-client.io-crd.yaml"), "expected generated CRD file")
}

func skipTestIfCommandFails(t *testing.T, name string, args ...string) {
	c := &cmdrunner.Command{
		Name: name,
		Args: args,
	}
	text, err := cmdrunner.DefaultCommandRunner(c)
	t.Logf("ran: %s\ngot: %s\n", c.CLI(), text)

	if err != nil {
		t.Logf("skipping the test as got %s\n", err.Error())
		t.SkipNow()
	}
}
