package release_test

import (
	"path/filepath"
	"testing"

	jxfake "github.com/jenkins-x/jx-api/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helm/release"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-helpers/pkg/kube"
	"github.com/jenkins-x/jx-helpers/pkg/kube/jxenv"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/yaml"
)

func TestStepHelmRelease(t *testing.T) {
	runner := &fakerunner.FakeRunner{}
	helmBin := "helm"

	ns := "jx"
	devEnv := jxenv.CreateDefaultDevEnvironment(ns)
	devEnv.Namespace = ns
	devEnv.Spec.Source.URL = "https://github.com/myorg/myrepo.git"

	requirements := config.NewRequirementsConfig()
	requirements.Cluster.ChartRepository = "http://bucketrepo/bucketrepo/charts/"
	data, err := yaml.Marshal(requirements)
	require.NoError(t, err, "failed to marshal requirements")
	devEnv.Spec.TeamSettings.BootRequirements = string(data)

	jxClient := jxfake.NewSimpleClientset(devEnv)

	_, o := release.NewCmdHelmRelease()
	o.HelmBinary = helmBin
	o.CommandRunner = runner.Run
	o.ChartsDir = filepath.Join("test_data", "charts")
	o.JXClient = jxClient
	o.Namespace = ns

	o.KubeClient = fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      kube.SecretJenkinsChartMuseum,
				Namespace: ns,
			},
			Data: map[string][]byte{
				"BASIC_AUTH_USER": []byte("myuser"),
				"BASIC_AUTH_PASS": []byte("mypwd"),
			},
		},
	)

	err = o.Run()
	require.NoError(t, err, "failed to run the command")

	for _, c := range runner.OrderedCommands {
		t.Logf("ran: %s\n", c.CLI())
	}
}
