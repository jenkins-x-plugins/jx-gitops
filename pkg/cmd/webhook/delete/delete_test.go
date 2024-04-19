package delete_test

import (
	"context"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/webhook/delete"
	"github.com/jenkins-x/go-scm/scm"
	scmFake "github.com/jenkins-x/go-scm/scm/driver/fake"
	jxv1 "github.com/jenkins-x/jx-api/v4/pkg/apis/jenkins.io/v1"
	jxfake "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

// Test Cases:
// 1. No Source repos
// 2. No hooks
// 3. Hook with dry run - see that it has not deleted it
// 4. Hook without dry run - see that is has deleted it
// 5. Test all delete flag

const (
	testOrg       = "jx-test-org"
	testRepo      = "jx-test-repo"
	testNamespace = "jx-test"
)

func createSr(create bool) *jxv1.SourceRepository {
	if !create {
		return &jxv1.SourceRepository{}
	}
	return &jxv1.SourceRepository{
		TypeMeta: metav1.TypeMeta{},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "jx-test",
		},
		Spec: jxv1.SourceRepositorySpec{
			Description:  "",
			Provider:     "https://fakegithub.com",
			Org:          testOrg,
			Repo:         testRepo,
			ProviderName: "fakegithub",
			ProviderKind: "fakegithub",
			URL:          "https://fakegithub.com/jx-test/jx-test",
			HTTPCloneURL: "https://fakegithub.com/jx-test/jx-test.git",
		},
	}
}

func TestWebhookDelete(t *testing.T) {
	testCases := []struct {
		description string
		sourceRepo  *jxv1.SourceRepository
		createhook  bool
		dryRun      bool
		deleteAll   bool
	}{
		{description: "No Source repositories", sourceRepo: createSr(false), createhook: false, dryRun: false, deleteAll: false},
		{description: "No hooks", sourceRepo: createSr(true), createhook: false, dryRun: false, deleteAll: false},
		{description: "Hook with dryrun - single webhook", sourceRepo: createSr(true), createhook: true, dryRun: true, deleteAll: false},
		{description: "Hook without dryrun - single webhook", sourceRepo: createSr(true), createhook: true, dryRun: false, deleteAll: false},
		{description: "Delete all webhooks - multiple webhooks", sourceRepo: createSr(true), createhook: true, dryRun: false, deleteAll: true},
	}
	for _, tc := range testCases {
		t.Log(tc.description)
		cmd, o := delete.NewCmdWebHookDelete()
		o.Namespace = testNamespace
		namespace := &v1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: testNamespace,
			},
		}

		env := &jxv1.Environment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "jx-test",
				Namespace: testNamespace,
			},
		}

		kubeClient := fake.NewSimpleClientset(namespace)
		o.KubeClient = kubeClient
		jxClient := jxfake.NewSimpleClientset(env, tc.sourceRepo)
		o.JXClient = jxClient
		scmClient, _ := scmFake.NewDefault()

		if tc.createhook {
			in := &scm.HookInput{
				Target: "http://example.com",
				Name:   "test",
			}

			_, _, err := scmClient.Repositories.CreateHook(context.Background(), scm.Join(testOrg, testRepo), in)
			if err != nil {
				t.Fatal(err)
			}
			if tc.deleteAll {
				in.Target = "http://example2.com"
				_, _, err := scmClient.Repositories.CreateHook(context.Background(), scm.Join(testOrg, testRepo), in)
				if err != nil {
					t.Fatal(err)
				}
			}
		}

		o.ScmClientFactory.ScmClient = scmClient

		o.Filter = "http://"
		o.DryRun = tc.dryRun
		err := cmd.Execute()
		assert.NoError(t, err)
		if tc.createhook {
			hooks, _, err := o.ScmClientFactory.ScmClient.Repositories.ListHooks(context.Background(), scm.Join(testOrg, testRepo), &scm.ListOptions{})
			if err != nil {
				t.Fatal(err)
			}
			if tc.dryRun {
				if len(hooks) == 0 {
					t.Fatal("Expected no deletion because dry run is enabled")
				}
			} else {
				if len(hooks) != 0 {
					t.Fatal("Expected deletion because dry run is disabled")
				}
			}

		}
	}
}
