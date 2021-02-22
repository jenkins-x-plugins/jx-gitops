package update_test

import (
	"context"
	"testing"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/webhook/update"

	"github.com/jenkins-x/go-scm/scm"
	fakejx "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/boot"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers/testjx"
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
				Name:      update.LighthouseHMACToken,
				Namespace: ns,
			},
			Data: map[string][]byte{
				"hmac": []byte("dummyhmactoken"),
			},
		},
	)

	requirements := jxcore.NewRequirementsConfig()
	requirements.Spec.Cluster.ChartRepository = "http://bucketrepo/bucketrepo/charts/"
	data, err := yaml.Marshal(requirements)
	require.NoError(t, err, "failed to marshal requirements")

	devEnv := jxenv.CreateDefaultDevEnvironment(ns)
	devEnv.Namespace = ns
	devEnv.Spec.Source.URL = "https://github.com/myorg/myrepo.git"
	devEnv.Spec.TeamSettings.BootRequirements = string(data)

	sr := testjx.CreateSourceRepository(ns, owner, repo, "fake", "https://fake.git")

	jxClient := fakejx.NewSimpleClientset(devEnv, sr)

	_, o := update.NewCmdWebHookVerify()
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

	sr, err = jxClient.JenkinsV1().SourceRepositories(ns).Get(context.TODO(), sr.Name, metav1.GetOptions{})
	require.NoError(t, err, "failed to lookup SourceRepository %s", sr.Name)
	testhelpers.AssertAnnotation(t, update.WebHookAnnotation, "true", sr.ObjectMeta, "for SourceRepository: "+sr.Name)
	t.Logf("SourceRepository %s has annotation %s = %s\n", sr.Name, update.WebHookAnnotation, sr.Annotations[update.WebHookAnnotation])
}
