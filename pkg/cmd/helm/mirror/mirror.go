package mirror

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx-gitops/pkg/ghpages"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/v3/pkg/httphelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/versionstream"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/repo"
)

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Escapes any {{ or }} characters in the YAML files so they can be included in a helm chart
`)

	cmdExample = templates.Examples(`
		# escapes any yaml files so they can be included in a helm chart 
		%s helm escape --dir myyaml
	`)
)

// Options the options for the command
type Options struct {
	scmhelpers.Factory
	Dir              string
	RepositoriesFile string
	Branch           string
	GitURL           string
	CommitMessage    string
	Excludes         []string
	NoPush           bool
	GitClient        gitclient.Interface
	CommandRunner    cmdrunner.CommandRunner
}

// NewCmdMirror creates a command object for the command
func NewCmdMirror() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "mirror",
		Short:   "Creates a helm mirror ",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory which contains the charts/repositories.yml file")
	cmd.Flags().StringVarP(&o.Branch, "branch", "b", "gh-pages", "the git branch to clone the repository")
	cmd.Flags().StringVarP(&o.GitURL, "url", "u", "", "the git URL of the repository to mirror the charts into")
	cmd.Flags().StringVarP(&o.CommitMessage, "message", "m", "chore: upgrade mirrored charts", "the commit message")
	cmd.Flags().StringArrayVarP(&o.Excludes, "exclude", "x", []string{"jenkins-x", "jx3"}, "the helm repositories to exclude from mirroring")

	o.Factory.AddFlags(cmd)
	return cmd, o
}

// Validate the arguments
func (o *Options) Validate() error {
	if o.GitURL == "" {
		return options.MissingOption("url")
	}
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
	}
	if o.GitClient == nil {
		o.GitClient = cli.NewCLIClient("", o.CommandRunner)
	}

	if o.GitToken == "" {
		if o.GitServerURL == "" {
			gitInfo, err := giturl.ParseGitURL(o.GitURL)
			if err != nil {
				return errors.Wrapf(err, "failed to parse git URL %s", o.GitURL)
			}
			o.GitServerURL = gitInfo.HostURL()
		}

		err := o.Factory.FindGitToken()
		if err != nil {
			return errors.Wrapf(err, "failed to find git token")
		}
		if o.GitToken == "" {
			return options.MissingOption("git-token")
		}
	}
	return nil
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate options")
	}
	prefixes, err := versionstream.GetRepositoryPrefixes(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to load quickstart repositories")
	}
	if prefixes == nil {
		return errors.Errorf("no chart repository prefixes")
	}
	if len(prefixes.Repositories) == 0 {
		return errors.Errorf("could not find charts/repositories.yml file in dir %s", o.Dir)
	}

	gitDir, err := ghpages.CloneGitHubPagesToDir(o.GitClient, o.GitURL, o.Branch, o.GitUsername, o.GitToken)
	if err != nil {
		return errors.Wrapf(err, "failed to clone the github pages repo %s branch %s", o.GitURL, o.Branch)
	}
	if gitDir == "" {
		return errors.Errorf("no github pages clone dir")
	}
	log.Logger().Infof("cloned github pages repository to %s", info(gitDir))

	for _, repo := range prefixes.Repositories {
		name := repo.Prefix
		if stringhelpers.StringArrayIndex(o.Excludes, name) >= 0 {
			continue
		}
		outDir := filepath.Join(gitDir, name)
		err = os.MkdirAll(outDir, files.DefaultDirWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to create dir %s", outDir)
		}

		err = o.MirrorRepository(outDir, repo.URLs)
		if err != nil {
			return errors.Wrapf(err, "failed to mirror repository %s", name)
		}
	}

	changes, err := gitclient.AddAndCommitFiles(o.GitClient, gitDir, o.CommitMessage)
	if err != nil {
		return errors.Wrapf(err, "failed to add and commit files")
	}
	if !changes {
		log.Logger().Infof("no changes")
		return nil
	}
	if o.NoPush {
		return nil
	}
	err = gitclient.Pull(o.GitClient, gitDir)
	if err != nil {
		return errors.Wrapf(err, "failed to push changes")
	}

	log.Logger().Infof("pushed changes to %s in branch %s", info(o.GitURL), info(o.Branch))
	return nil
}

// MirrorRepository downloads the index yaml and all the referenced charts to the given directory
func (o *Options) MirrorRepository(dir string, urls []string) error {
	for _, u := range urls {
		path := filepath.Join(dir, "index.yaml")
		indexURL := stringhelpers.UrlJoin(u, "index.yaml")
		err := downloadURLToFile(indexURL, path)
		if err != nil {
			log.Logger().Warnf("failed to download index for %s", indexURL)
			continue
		}

		idx, err := repo.LoadIndexFile(path)
		if err != nil {
			log.Logger().Warnf("failed to load index file at %s", path)
			return nil
		}

		log.Logger().Infof("downloaded %s", info(path))
		err = o.DownloadIndex(idx, u, dir)
		if err != nil {
			return errors.Wrapf(err, "failed to download index for %s", dir)
		}
	}
	return nil
}

func (o *Options) DownloadIndex(idx *repo.IndexFile, u, dir string) error {
	for _, v := range idx.Entries {
		for _, cv := range v {
			for _, name := range cv.URLs {
				path := filepath.Join(dir, name)
				exists, err := files.FileExists(path)
				if err != nil {
					return errors.Wrapf(err, "failed to check for path %s", path)
				}
				if exists {
					continue
				}

				fileURL := stringhelpers.UrlJoin(u, name)
				err = downloadURLToFile(fileURL, path)
				if err != nil {
					log.Logger().Warnf("failed to download %s", fileURL)
					continue
				}
				log.Logger().Infof("downloaded %s", info(path))
			}
		}
	}
	return nil
}

func downloadURLToFile(u string, path string) error {
	dir := filepath.Dir(path)
	err := os.MkdirAll(dir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create dir %s", dir)
	}

	client := httphelpers.GetClient()
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to create http request for %s", u)
	}

	resp, err := client.Do(req)
	if err != nil {
		if resp != nil {
			return errors.Wrapf(err, "failed to GET endpoint %s with status %s", u, resp.Status)
		}
		return errors.Wrapf(err, "failed to GET endpoint %s", u)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrapf(err, "failed to read response from %s", u)
	}

	err = ioutil.WriteFile(path, body, files.DefaultFileWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", path)
	}
	return nil
}
