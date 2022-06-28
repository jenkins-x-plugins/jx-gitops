package clone_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/git/clone"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGitClone(t *testing.T) {
	_, o := clone.NewCmdGitClone()

	runner := &fakerunner.FakeRunner{}
	o.CommandRunner = runner.Run
	o.UserEmail = "fakeuser@googlegroups.com"
	o.DisableInClusterTest = true

	ns := "jx"

	o.Namespace = ns
	o.KubeClient = fake.NewSimpleClientset(
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "jx-boot",
				Namespace: ns,
			},
			Data: map[string][]byte{
				"url":             []byte("https://github.com/myorg/myrepo.git"),
				"username":        []byte("myuser"),
				"password":        []byte("mypwd"),
				"gitInitCommands": []byte("echo hey"),
			},
		},
	)
	tmpDir := t.TempDir()
	o.OutputFile = filepath.Join(tmpDir, "git-credentials")
	o.Dir = tmpDir

	t.Logf("creating git credentials file %s", o.OutputFile)

	err := o.Run()
	require.NoError(t, err, "failed to run git setup")

	runner.ExpectResults(t,
		fakerunner.FakeResult{
			CLI: "git config --global --add user.name myuser",
		},
		fakerunner.FakeResult{
			CLI: "git config --global --add user.email fakeuser@googlegroups.com",
		},
		fakerunner.FakeResult{
			CLI: "git config --global credential.helper store",
		},
		fakerunner.FakeResult{
			CLI: "sh -c echo hey",
		},
		fakerunner.FakeResult{
			CLI: "git clone https://github.com/myorg/myrepo.git " + filepath.Join(tmpDir, "source"),
		},
	)

	testhelpers.AssertTextFilesEqual(t, filepath.Join("testdata", "expected.txt"), o.OutputFile, "generated git credentials file")
}
