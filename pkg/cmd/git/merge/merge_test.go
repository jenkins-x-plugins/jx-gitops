package merge_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/git/merge"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitMerge(t *testing.T) {
	masterSha := ""
	branchBSha := ""
	branchCSha := ""
	dir := ""

	g := cli.NewCLIClient("", nil)

	testCases := []struct {
		name  string
		init  func(*merge.Options)
		check func()
	}{
		{
			name: "explicit-arguments",
			init: func(o *merge.Options) {
				o.SHAs = []string{branchBSha}
				o.BaseBranch = "master"
				o.BaseSHA = masterSha
			},
			check: func() {
				assert.Equal(t, branchBSha, readHeadSHA(t, dir), "should have merged head SHA")
			},
		},
		{
			name: "with-pullrefs",
			init: func(o *merge.Options) {
				o.PullRefs = fmt.Sprintf("master:%s,b:%s", masterSha, branchBSha)
			},
			check: func() {
				assert.Equal(t, branchBSha, readHeadSHA(t, dir), "should be on the right sha")
			},
		},
		{
			name: "multiple-shas-in-pullrefs",
			init: func(o *merge.Options) {
				o.PullRefs = fmt.Sprintf("master:%s,c:%s,b:%s", masterSha, branchCSha, branchBSha)
			},
			check: func() {
			},
		},
	}

	tmpDir, err := ioutil.TempDir("", "")
	require.NoError(t, err, "failed to create tmp dir")

	for _, tc := range testCases {
		name := tc.name
		dir = filepath.Join(tmpDir, name)

		err := os.MkdirAll(dir, files.DefaultDirWritePermissions)
		require.NoError(t, err, "failed to create dir %s", dir)

		err = gitclient.Init(g, dir)
		require.NoError(t, err, "failed to git init for %s", name)

		requireWritefile(t, dir, "a.txt", "a")
		requireGitAdd(t, g, dir)
		masterSha = requireCommit(t, g, dir, "a commit")

		requireNewBranch(t, g, dir, "b")
		requireWritefile(t, dir, "b.txt", "b")
		requireGitAdd(t, g, dir)
		branchBSha = requireCommit(t, g, dir, "b commit")

		requireGit(t, g, dir, "checkout", "master")
		requireNewBranch(t, g, dir, "c")
		requireWritefile(t, dir, "c.txt", "c")
		requireGitAdd(t, g, dir)
		branchCSha = requireCommit(t, g, dir, "c commit")

		requireGit(t, g, dir, "checkout", "master")
		_, o := merge.NewCmdGitMerge()

		assert.Equal(t, masterSha, readHeadSHA(t, dir), "should be on the right head SHA for %s", name)

		if tc.init != nil {
			tc.init(o)
		}
		o.Dir = dir
		err = o.Run()
		require.NoError(t, err, "running merge for test %s", name)

		if tc.check != nil {
			tc.check()
		}
	}
}

func requireWritefile(t *testing.T, dir string, name string, contents string) {
	path := filepath.Join(dir, name)
	err := ioutil.WriteFile(path, []byte(contents), files.DefaultFileWritePermissions)
	require.NoError(t, err, "failed to write file %s", path)
}

func requireGitAdd(t *testing.T, g gitclient.Interface, dir string) {
	err := gitclient.Add(g, dir, "*")
	require.NoError(t, err, "failed to git add in dir %s, dir")
}

func requireCommit(t *testing.T, g gitclient.Interface, dir string, message string) string {
	_, err := g.Command(dir, "commit", "-m", message, "--no-gpg-sign")
	require.NoError(t, err, "failed to git commit")
	return readHeadSHA(t, dir)
}

func requireNewBranch(t *testing.T, g gitclient.Interface, dir string, branch string) {
	_, err := g.Command(dir, "checkout", "-b", branch)
	require.NoError(t, err, "failed to create branch %s", branch)
}

func requireGit(t *testing.T, g gitclient.Interface, dir string, args ...string) {
	_, err := g.Command(dir, args...)
	require.NoError(t, err, "failed to perform git %s", strings.Join(args, " "))
}

// readHeadSHA asserts we have the current head sha
func readHeadSHA(t *testing.T, dir string) string {
	path := filepath.Join(dir, ".git", "HEAD")
	data, err := ioutil.ReadFile(path)
	require.NoError(t, err, "failed to load file %s", path)

	var sha string
	if strings.HasPrefix(string(data), "ref:") {
		headRef := strings.TrimPrefix(string(data), "ref: ")
		headRef = strings.Trim(headRef, "\n")
		sha = readRef(t, dir, headRef)
	} else {
		sha = string(data)
	}
	return sha
}

// readRef reads the commit SHA of the specified ref. Needs to be of the form /refs/heads/<name>.
func readRef(t *testing.T, repoDir string, name string) string {
	path := filepath.Join(repoDir, ".git", name)
	data, err := ioutil.ReadFile(path)
	require.NoError(t, err, "failed to read path %s", path)
	return strings.Trim(string(data), "\n")
}
