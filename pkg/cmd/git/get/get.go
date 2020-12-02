package get

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jenkins-x/go-scm/scm"
	jxc "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	cmdLong = templates.LongDesc(`
		Gets a file from a git repository or environment git repository
`)

	cmdExample = templates.Examples(`
		%s git get --file jx-values.yaml --dev dev 
	`)

	info = termcolor.ColorInfo
)

// Options the options for the command
type Options struct {
	scmhelpers.Options

	Env            string
	FromRepository string
	Path           string
	To             string
	Ref            string
	Namespace      string
	JXClient       jxc.Interface
}

// NewCmdGitGet creates a command object for the command
func NewCmdGitGet() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "get",
		Short:   "Gets a file from a git repository or environment git repository",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.Options.AddFlags(cmd)

	cmd.Flags().StringVarP(&o.FromRepository, "from", "", "", "the git repository of the form owner/name to find the file")
	cmd.Flags().StringVarP(&o.Env, "env", "e", "", "the name of the Environment to find the git repository URL")
	cmd.Flags().StringVarP(&o.Path, "file", "f", "", "the file in the git repository")
	cmd.Flags().StringVarP(&o.To, "to", "", "", "the destination of the file. If not specified defaults to the path")
	cmd.Flags().StringVarP(&o.Ref, "ref", "", "master", "the git reference (branch, tag or SHA) to query the file")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate options")
	}

	ctx := context.Background()
	sha := o.Ref
	path := o.Path
	repo := o.FromRepository
	c, r, err := o.ScmClient.Contents.Find(ctx, repo, path, sha)
	if err != nil {
		if r != nil && r.Status == 404 {
			return errors.Errorf("no file %s in repo %s for ref %s", path, repo, sha)
		}
		return errors.Wrapf(err, "failed to find file %s in repo %s with ref %s status %d", path, repo, sha, r.Status)
	}
	to := o.To
	if to == "" {
		to = filepath.Join(o.Dir, path)
	}
	dir := filepath.Dir(to)
	err = os.MkdirAll(dir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create dir %s", dir)
	}
	err = ioutil.WriteFile(to, c.Data, files.DefaultFileWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", to)
	}
	log.Logger().Infof("saved file %s from repository %s ref %s", info(to), info(repo), info(sha))
	return nil
}

// Validate validates the inputs are valid
func (o *Options) Validate() error {
	if o.Options.CommandRunner == nil {
		o.Options.CommandRunner = cmdrunner.QuietCommandRunner
	}
	var err error
	if o.FromRepository == "" && o.Env != "" {
		err = o.findEnvironmentRepository()
		if err != nil {
			return errors.Wrapf(err, "failed to find repository of environment %s", o.Env)
		}
	}

	err = o.Options.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate repository options")
	}

	if o.FromRepository == "" {
		return options.MissingOption("from")
	}
	if o.Path == "" {
		return options.MissingOption("file")
	}

	return nil
}

func (o *Options) findEnvironmentRepository() error {
	var err error
	o.JXClient, o.Namespace, err = jxclient.LazyCreateJXClientAndNamespace(o.JXClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to create jx client")
	}

	envName := o.Env
	env, err := o.JXClient.JenkinsV1().Environments(o.Namespace).Get(context.TODO(), envName, metav1.GetOptions{})
	if err != nil {
		log.Logger().Warnf("could not find environment %s in namespace %s ", envName, o.Namespace)
		return errors.Wrapf(err, "failed to load Environment %s in namespace %s", envName, o.Namespace)
	}
	gitURL := env.Spec.Source.URL
	log.Logger().Infof("environment %s in namespace %s has git URL %s", info(envName), info(o.Namespace), info(gitURL))

	if gitURL == "" {
		return errors.Errorf("no env.Spec.Source.URL for environment %s", envName)
	}

	o.SourceURL = gitURL

	gitInfo, err := giturl.ParseGitURL(gitURL)
	if err != nil {
		return errors.Wrapf(err, "failed to parse environment %s git URL %s", env.Name, gitURL)
	}

	o.FromRepository = scm.Join(gitInfo.Organisation, gitInfo.Name)
	o.Owner = gitInfo.Organisation
	o.Repository = gitInfo.Name
	return nil
}
