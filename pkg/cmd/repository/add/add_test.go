package add_test

import (
	"path/filepath"
	"testing"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/repository/add"
	jxfake "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned/fake"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/testhelpers/testjx"
	"github.com/stretchr/testify/require"
)

func TestRepositoryAdd(t *testing.T) {
	testCases := []struct {
		owner, repo, provider, kind string
		jenkins, scheduler          string
	}{
		{
			owner:    "jenkins-x",
			repo:     "myjenkins",
			provider: "https://github.com",
			jenkins:  "jenkins1",
		},
		{
			owner:    "something",
			repo:     "mygitlab",
			provider: "https://mygitlab.com",
			kind:     "gitlab",
		},
		{
			owner:    "jenkins-x",
			repo:     "anewthingy",
			provider: "https://github.com",
		},
	}
	rootTmpDir := t.TempDir()

	ns := "jx"
	for _, tc := range testCases {
		name := tc.repo
		sourceData := filepath.Join("test_data", name)

		tmpDir := filepath.Join(rootTmpDir, name)

		t.Logf("running test %s in %s", name, tmpDir)

		err := files.CopyDirOverwrite(sourceData, tmpDir)
		require.NoError(t, err, "failed to copy from %s to %s", sourceData, tmpDir)

		sr := testjx.CreateSourceRepository(ns, tc.owner, tc.repo, tc.kind, tc.provider)

		_, o := add.NewCmdAddRepository()
		o.Dir = tmpDir
		o.Args = []string{stringhelpers.UrlJoin(tc.provider, tc.owner, tc.repo+".git")}
		o.JXClient = jxfake.NewSimpleClientset(sr)
		o.Namespace = ns
		o.Jenkins = tc.jenkins

		err = o.Run()
		require.NoError(t, err, "failed to run")

		testhelpers.AssertTextFilesEqual(t, filepath.Join(tmpDir, "expected.yaml"), filepath.Join(tmpDir, ".jx", "gitops", "source-config.yaml"), "generated source config")
	}
}
