package helm_test

import (
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/helm"
	"github.com/jenkins-x/jx-gitops/pkg/fakes/fakegit"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/api/extensions/v1beta1"
	"sigs.k8s.io/yaml"
)

func TestStepHelmTemplate(t *testing.T) {
	_, o := helm.NewCmdHelmTemplate()

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "failed to create tmp dir")

	name := "mychart"
	o.ReleaseName = name
	o.Chart = filepath.Join("test_data", name)
	o.OutDir = tmpDir
	o.BatchMode = true
	o.Gitter = fakegit.NewGitFakeClone()
	o.ValuesFile = filepath.Join(tmpDir, "template-values.yaml")

	err = o.Run()
	require.NoError(t, err, "failed to run the command")

	templateDir := tmpDir
	require.DirExists(t, templateDir)

	t.Logf("generated templates to %s", templateDir)

	assert.FileExists(t, filepath.Join(templateDir, "deployment.yaml"))
	assert.FileExists(t, filepath.Join(templateDir, "service.yaml"))

	ingressFile := filepath.Join(templateDir, "ingress.yaml")
	require.FileExists(t, ingressFile)

	data, err := ioutil.ReadFile(ingressFile)
	require.NoError(t, err, "failed to load YAML from %s", ingressFile)

	ing := &v1beta1.Ingress{}
	err = yaml.Unmarshal(data, ing)
	require.NoError(t, err, "failed to parse YAML from %s", ingressFile)

	require.Equal(t, 1, len(ing.Spec.TLS), "ing.Spec.TLS")
	require.Equal(t, 1, len(ing.Spec.Rules), "ing.Spec.Rules")

	tls := ing.Spec.TLS[0]
	expectedHost := "mychart.cluster.local"
	assert.Equal(t, []string{expectedHost}, tls.Hosts, "ing.Spec.TLS[0].Hosts")

	rule := ing.Spec.Rules[0]
	assert.Equal(t, expectedHost, rule.Host, "ing.Spec.Rules[0].Host")
}
