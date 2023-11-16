package apply

import (
	"testing"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner/fakerunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/stretchr/testify/assert"
)

type testcase struct {
	PullRequest bool
	ShouldRegen bool
	Merge       bool
}

/*
A test git repo is checked in as a tarball in testdata. Each commit in the repo together with whether a regen should be
done if a push event arrived for that commit is listed below. Since the exact command issued depends on whether it's a
merge and if it's in a PR that is specified as well. To make it slightly less verbose I rely on false being the default
for bool.
*/
func TestRun(t *testing.T) {
	var testcases = map[string]testcase{
		"4d028367b24b7809480ba827a758b4c75d1085ad": {
			ShouldRegen: true,
		},
		"7c6e4dd1482d4c1965b73bfc9aecbe2ed61f9213": {
			ShouldRegen: true,
		},
		"c4c917e6870b646e423979848f8ce17694be300c": {
			ShouldRegen: false,
		},
		"5a2d23a558f9709c84ecb4ce73cb63465a878024": {
			PullRequest: true,
			ShouldRegen: true,
		},
		"3aa660eccc633686ba4f57175bd799888aad6a52": {
			ShouldRegen: true,
			Merge:       true,
		},
		"8a776ad1cc02dac4c840156ce5463d03f67f6d98": {},
		"103b6d7c756f1a1b55d76831cd127c08a6dc6ab8": {
			ShouldRegen: true,
			PullRequest: true,
		},
		"928fc962a680bcc248341af02ec4988cb3e7dc56": {
			PullRequest: true,
		},
		"95434f2de92dbae2bd8421a48836b59f24e5d444": {
			Merge: true,
		},
		"3ae5989bb7b04d244a68fece41167e1e5298ef19": {
			PullRequest: true,
			ShouldRegen: true,
		},
		"c89373adad0c6805fdf6850332d3a577455a9150": {
			PullRequest: true,
		},
		"7396b58ca141a72ef644a32c935a464a2ed960f5": {
			Merge: true,
		},
		"0ee9d85ffd453bbdeac2d7b2d2105886a6ed3afe": {
			ShouldRegen: true,
		},
		"08569d51acdf2e9d95c4353aaf201d308f385ecf": {
			PullRequest: true,
			ShouldRegen: true,
		},
		"23dd9fb4eb844ec69d718e962ea47c95d28d158c": {
			PullRequest: true,
		},
		"f8bcc7ec957da69e1318a6b2d0841878e63ec9e2": {
			ShouldRegen: true,
			Merge:       true,
		},
		"5936fa8287518cdb839d2c9a7442953161229275": {
			PullRequest: true,
			ShouldRegen: true,
		},
		"b219b3d0bd8847374606727726a5dc8d5e0f5697": {
			PullRequest: true,
		},
		"cbd994c27c385cfc2d9fdd77d11b8fa34e76f8bf": {
			ShouldRegen: true,
			Merge:       true,
		},
		"f1dcbd3f834b40a3cd597278475c6d26645d4b34": {},
		"f0873b49e8bf96c13368e2ecf773ac4195d4afa5": {
			PullRequest: true,
			ShouldRegen: true,
		},
		"c801731f74686677d168003c61cfd7af0b213fb1": {
			PullRequest: true,
		},
		"d30b050abf2ca5b04f5d1991e685f84bad0880cd": {
			ShouldRegen: true,
			Merge:       true,
		},
		"11653f46e4d8bb4541eee91e8ee6f75dc01d0b96": {},
		"e58fc10b0cfd36c39668810c9fef8c0fbae40088": {
			PullRequest: true,
			ShouldRegen: true,
		},
		"303cb834578ec30783ff2b8e9de9aaf25f04704b": {
			PullRequest: true,
		},
		"55ff686b3afa342a90c632d5b41419eb9a9ce11b": {
			Merge: true,
		},
	}

	dir := t.TempDir()

	assert.NoError(t, files.UnTargzAll("testdata/repo.tgz", dir))

	r, err := git.PlainOpen(dir)
	assert.NoError(t, err)

	tree, err := r.Worktree()
	assert.NoError(t, err)

	log, err := r.Log(&git.LogOptions{All: true, Order: git.LogOrderCommitterTime})
	assert.NoError(t, err)

	err = log.ForEach(func(commit *object.Commit) error {
		t.Run(commit.Message, func(t *testing.T) {

			err = tree.Reset(&git.ResetOptions{
				Commit: commit.Hash,
				Mode:   git.HardReset,
			})
			assert.NoError(t, err)

			test, ok := testcases[commit.Hash.String()]
			assert.Truef(t, ok, "Commit %s is missing from test cases", commit.Hash.String())

			fakeRunner := fakerunner.FakeRunner{}

			o := Options{
				Dir:           dir,
				PullRequest:   test.PullRequest,
				CommandRunner: fakeRunner.Run,
				IsNewCluster:  false,
				repo:          r,
			}
			err = o.Run()
			assert.NoError(t, err)

			if test.ShouldRegen {
				if test.PullRequest {
					assert.Len(t, fakeRunner.OrderedCommands, 1)
					assert.NotEmpty(t, fakeRunner.OrderedCommands[0].Args)
					assert.Equal(t, "pr-regen", fakeRunner.OrderedCommands[0].Args[0])
				} else {
					assert.Len(t, fakeRunner.OrderedCommands, 3)
					assert.NotEmpty(t, fakeRunner.OrderedCommands[0].Args)
					assert.Equal(t, "regen-phase-1", fakeRunner.OrderedCommands[0].Args[0])
					assert.NotEmpty(t, fakeRunner.OrderedCommands[1].Args)
					assert.Equal(t, "regen-phase-2", fakeRunner.OrderedCommands[1].Args[0])
					assert.NotEmpty(t, fakeRunner.OrderedCommands[2].Args)
					assert.Equal(t, "regen-phase-3", fakeRunner.OrderedCommands[2].Args[0])
				}
			} else if test.Merge {
				assert.Len(t, fakeRunner.OrderedCommands, 1)
				assert.NotEmpty(t, fakeRunner.OrderedCommands[0].Args)
				assert.Equal(t, "regen-none", fakeRunner.OrderedCommands[0].Args[0])
			} else {
				assert.Empty(t, fakeRunner.OrderedCommands)
			}
		})

		return nil
	})
	assert.NoError(t, err)
}
