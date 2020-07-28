package template_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/apps/template"
	"github.com/jenkins-x/jx-gitops/pkg/fakekpt"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/pkg/testhelpers"
	"github.com/jenkins-x/jx-helpers/pkg/yamls"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func TestStepJxAppsTemplate(t *testing.T) {
	secretsYaml := filepath.Join("test_data", "input", "secrets.yaml")
	require.FileExists(t, secretsYaml)

	_, o := template.NewCmdJxAppsTemplate()

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "failed to create tmp dir")

	testSrcDir := filepath.Join("test_data", "input")

	o.Dir = filepath.Join(tmpDir, "src")
	err = files.CopyDirOverwrite(testSrcDir, o.Dir)
	require.NoError(t, err, "failed to copy %s to %s", testSrcDir, o.Dir)

	templateDir := filepath.Join(tmpDir, "config-root")

	t.Logf("resolving source in %s", o.Dir)
	t.Logf("generated templates to %s", templateDir)

	o.OutDir = templateDir
	o.Options.VersionStreamDir = filepath.Join("test_data", "versionstream")

	o.TemplateValuesFiles = []string{secretsYaml}
	runner := &fakerunner.FakeRunner{
		CommandRunner: func(c *cmdrunner.Command) (string, error) {
			if c.Name == "clone" && len(c.Args) > 0 {
				// lets really git clone but then fake out all other commands
				return cmdrunner.DefaultCommandRunner(c)
			}
			if c.Name == "kpt" {
				srcDir := o.Options.VersionStreamDir
				destDir := o.Dir
				return fakekpt.FakeKpt(t, c, srcDir, destDir)
			}
			return "", nil
		},
	}
	o.Options.CommandRunner = runner.Run
	o.TemplateOptions.Gitter = cli.NewCLIClient("", runner.Run)

	err = o.Run()
	require.NoError(t, err, "failed to run the command")

	assert.FileExists(t, filepath.Join(templateDir, "jx", "bucketrepo", "deployment.yaml"))
	assert.FileExists(t, filepath.Join(templateDir, "foo", "external-dns", "deployment.yaml"))
	assert.FileExists(t, filepath.Join(templateDir, "foo", "external-dns", "service.yaml"))
	assert.FileExists(t, filepath.Join(templateDir, "foo", "external-dns", "clusterrolebinding.yaml"))

	tektonSAFile := filepath.Join(templateDir, "jx", "tekton", "251-bot-serviceaccount.yaml")
	assert.FileExists(t, tektonSAFile)

	tektonSA := &corev1.ServiceAccount{}
	err = yamls.LoadFile(tektonSAFile, tektonSA)

	require.NoError(t, err, "failed to load file %s", tektonSAFile)
	message := fmt.Sprintf("tekton SA for file %s", tektonSAFile)

	testhelpers.AssertAnnotation(t, "iam.gke.io/gcp-service-account", "mycluster-tk@myproject.iam.gserviceaccount.com", tektonSA.ObjectMeta, message)

	// verify we generated the chart and its dependencies
	assert.FileExists(t, filepath.Join(templateDir, "jx", "jxboot-helmfile-resources", "docker-cfg-secret.yaml"))
	assert.FileExists(t, filepath.Join(templateDir, "jx", "jxboot-helmfile-resources", "controllerbuild", "serviceaccount.yaml"))

	externalSecretsSAFile := filepath.Join(templateDir, "external-secrets", "kubernetes-external-secrets", "serviceaccount.yaml")
	assert.FileExists(t, externalSecretsSAFile)

	externalSecretsSA := &corev1.ServiceAccount{}
	err = yamls.LoadFile(externalSecretsSAFile, externalSecretsSA)

	require.NoError(t, err, "failed to load file %s", externalSecretsSAFile)
	message = fmt.Sprintf("external secrets SA for file %s", externalSecretsSAFile)

	testhelpers.AssertAnnotation(t, "iam.gke.io/gcp-service-account", "mycluster-es@myproject.iam.gserviceaccount.com", externalSecretsSA.ObjectMeta, message)
}
