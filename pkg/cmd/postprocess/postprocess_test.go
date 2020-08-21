package postprocess_test

import (
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/postprocess"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner/fakerunner"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestPostProcess(t *testing.T) {
	_, o := postprocess.NewCmdPostProcess()

	ns := "default"
	o.KubeClient = fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      postprocess.DefaultSecretName,
				Namespace: ns,
			},
			Data: map[string][]byte{
				"commands": []byte(`kubectl annotation sa cheese hello="world"`),
			},
		},
	)
	runner := &fakerunner.FakeRunner{}
	o.CommandRunner = runner.Run

	err := o.Run()
	require.NoError(t, err, "failed to run command")

	runner.ExpectResults(t,
		fakerunner.FakeResult{
			CLI: `sh -c kubectl annotation sa cheese hello="world"`,
		},
	)
}

func TestPostProcessDoesNotFailWithNoSecret(t *testing.T) {
	_, o := postprocess.NewCmdPostProcess()

	o.KubeClient = fake.NewSimpleClientset()
	runner := &fakerunner.FakeRunner{}
	o.CommandRunner = runner.Run

	err := o.Run()
	require.NoError(t, err, "failed to run command")
}
