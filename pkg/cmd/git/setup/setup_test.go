package setup_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/git/setup"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestGitSetup(t *testing.T) {
	testCases := []struct {
		username     string
		password     string
		expectedPath string
	}{
		{
			username:     "myuser",
			password:     "mypwd",
			expectedPath: filepath.Join("testdata", "expected.txt"),
		},
		{
			username:     "myuser",
			password:     "my/pwd",
			expectedPath: filepath.Join("testdata", "expected2.txt"),
		},
	}

	for _, tc := range testCases {
		_, o := setup.NewCmdGitSetup()

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
					"url":      []byte("https://github.com/myorg/myrepo.git"),
					"username": []byte(tc.username),
					"password": []byte(tc.password),
				},
			},
		)
		tmpFile, err := os.CreateTemp("", "")
		require.NoError(t, err, "failed to create temp flie")
		o.OutputFile = tmpFile.Name()

		t.Logf("creating git credentials file %s", o.OutputFile)

		// Unset GITHUB_TOKEN for the duration of the test
		os.Unsetenv("GITHUB_TOKEN")

		err = o.Run()
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
		)

		testhelpers.AssertTextFilesEqual(t, tc.expectedPath, o.OutputFile, "generated git credentials file")
	}
}

func TestGitSetupWithOperatorNamespace(t *testing.T) {
	_, o := setup.NewCmdGitSetup()

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
				Namespace: "jx-git-operator",
			},
			Data: map[string][]byte{
				"url":      []byte("https://github.com/myorg/myrepo.git"),
				"username": []byte("myuser"),
				"password": []byte("mypwd"),
			},
		},
	)
	tmpFile, err := os.CreateTemp("", "")
	require.NoError(t, err, "failed to create temp flie")
	o.OutputFile = tmpFile.Name()

	t.Logf("creating git credentials file %s", o.OutputFile)

	err = o.Run()
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
	)

	testhelpers.AssertTextFilesEqual(t, filepath.Join("testdata", "expected.txt"), o.OutputFile, "generated git credentials file")
}
