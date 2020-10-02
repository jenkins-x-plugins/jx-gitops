package verify_test

import (
	"context"
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	fakejx "github.com/jenkins-x/jx-api/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/webhook/verify"
	"github.com/jenkins-x/jx-helpers/pkg/boot"
	"github.com/jenkins-x/jx-helpers/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/pkg/testhelpers"
	"github.com/jenkins-x/jx-helpers/pkg/testhelpers/testjx"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
	"sigs.k8s.io/yaml"
)

func TestWebhookVerify(t *testing.T) {
	ns := "jx"
	owner := "myorg"
	repo := "myrepo"
	fullName := scm.Join(owner, repo)

	kubeClient := fake.NewSimpleClientset(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: ns,
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      boot.SecretName,
				Namespace: ns,
			},
			Data: map[string][]byte{
				"url":      []byte("https://fake.git/myorg/myrepo.git"),
				"username": []byte("myuser"),
				"password": []byte("mypwd"),
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      verify.LighthouseHMACToken,
				Namespace: ns,
			},
			Data: map[string][]byte{
				"hmac": []byte("dummyhmactoken"),
			},
		},
	)

	requirements := config.NewRequirementsConfig()
	requirements.Cluster.ChartRepository = "http://bucketrepo/bucketrepo/charts/"
	data, err := yaml.Marshal(requirements)
	require.NoError(t, err, "failed to marshal requirements")

	devEnv := jxenv.CreateDefaultDevEnvironment(ns)
	devEnv.Namespace = ns
	devEnv.Spec.Source.URL = "https://github.com/myorg/myrepo.git"
	devEnv.Spec.TeamSettings.BootRequirements = string(data)

	sr := testjx.CreateSourceRepository(ns, owner, repo, "fake", "https://fake.git")

	jxClient := fakejx.NewSimpleClientset(devEnv, sr)

	_, o := verify.NewCmdWebHookVerify()
	o.Namespace = ns
	o.KubeClient = kubeClient
	o.JXClient = jxClient
	o.ScmClientFactory.GitToken = "dummytoken"

	err = o.Run()
	require.NoError(t, err, "failed to run")

	hooks, _, err := o.ScmClientFactory.ScmClient.Repositories.ListHooks(context.Background(), fullName, scm.ListOptions{})
	require.NoError(t, err, "failed listing webhooks for repo %s", fullName)
	require.NotEmpty(t, hooks, "should have created a webbook for repository %s", fullName)

	for _, h := range hooks {
		t.Logf("found hook %s for %s with events %#v\n", h.ID, h.Target, h.Events)
	}

	sr, err = jxClient.JenkinsV1().SourceRepositories(ns).Get(sr.Name, metav1.GetOptions{})
	require.NoError(t, err, "failed to lookup SourceRepository %s", sr.Name)
	testhelpers.AssertAnnotation(t, verify.WebHookAnnotation, "true", sr.ObjectMeta, "for SourceRepository: "+sr.Name)
	t.Logf("SourceRepository %s has annotation %s = %s\n", sr.Name, verify.WebHookAnnotation, sr.Annotations[verify.WebHookAnnotation])
}
