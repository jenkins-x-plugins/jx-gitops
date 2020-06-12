package fakegit

import "github.com/jenkins-x/jx/pkg/gits"

// GitFakeClone struct for the fake git
type GitFakeClone struct {
	gits.GitFake
}

// NewGitFakeClone a fake Gitter but implements cloning
func NewGitFakeClone() gits.Gitter {
	f := &GitFakeClone{}
	f.Changes = true
	f.Commits = []gits.GitCommit{
		{
			SHA:     "mysha1234",
			Message: "some commit",
			Branch:  "master",
		},
	}
	return f
}

func (f *GitFakeClone) Clone(url string, directory string) error {
	return gits.NewGitCLI().Clone(url, directory)
}
