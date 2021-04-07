package release_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/fakerunners"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helm/release"
	jxfake "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/yaml"
)

func TestStepHelmRelease(t *testing.T) {
	runner := fakerunners.NewFakeRunnerWithGitClone()
	helmBin := "helm"

	ns := "jx"
	devEnv := jxenv.CreateDefaultDevEnvironment(ns)
	devEnv.Namespace = ns
	devEnv.Spec.Source.URL = "https://github.com/jx3-gitops-repositories/jx3-kubernetes.git"

	requirements := jxcore.NewRequirementsConfig()
	requirements.Spec.Cluster.ChartRepository = "http://bucketrepo/bucketrepo/charts/"
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
	o.Version = "1.2.3"

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

func TestStepHelmReleaseWithArtifactory(t *testing.T) {
	runner := fakerunners.NewFakeRunnerWithGitClone()
	helmBin := "helm"

	ns := "jx"
	devEnv := jxenv.CreateDefaultDevEnvironment(ns)
	devEnv.Namespace = ns
	devEnv.Spec.Source.URL = "https://github.com/jx3-gitops-repositories/jx3-kubernetes.git"

	requirements := jxcore.NewRequirementsConfig()
	requirements.Spec.Cluster.ChartRepository = "http://bucketrepo/bucketrepo/charts/"
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
	o.Version = "2.3.4"
	o.Artifactory = true

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
