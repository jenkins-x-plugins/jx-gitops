package jx_apps_test

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/jx_apps"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner/fakerunner"
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

	_, o := jx_apps.NewCmdJxAppsTemplate()

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "failed to create tmp dir")

	o.Dir = filepath.Join("test_data", "input")
	o.OutDir = tmpDir
	o.VersionStreamDir = filepath.Join("test_data", "versionstream")

	o.TemplateValuesFiles = []string{secretsYaml}
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
