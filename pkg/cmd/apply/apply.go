package apply

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Performs a gitops regeneration and apply on a cluster git repository

		If the last commit was a merge from a pull request the regeneration is skipped, unless the cluster is new.

		Also the process detects if an ingress has changed (or similar changes) and retriggers another regeneration which typically is only required when installing for the first time or if no explicit domain name is being used and the LoadBalancer service has been removed.
`)

	cmdExample = templates.Examples(`
		# performs a regeneration and apply
		%s apply
	`)
)

// Options the options for the command
type Options struct {
	Dir              string
	PullRequest      bool
	GitClient        gitclient.Interface
	CommandRunner    cmdrunner.CommandRunner
	GitCommandRunner cmdrunner.CommandRunner
	IsNewCluster     bool
}

// NewCmdApply creates a command object for the command
func NewCmdApply() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "apply",
		Short:   "Performs a GitOps regeneration and apply on a cluster git repository",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to the git and make commands")
	cmd.Flags().BoolVarP(&o.PullRequest, "pull-request", "", false, "specifies to apply the pull request contents into the PR branch")
	return cmd, o
}

// Validate validates the setup
func (o *Options) Validate() error {
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.QuietCommandRunner
	}
	if o.GitClient == nil {
		o.GitClient = cli.NewCLIClient("", o.CommandRunner)
	}
	o.IsNewCluster = o.isNewCluster()
	return nil
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate")
	}

	lastCommitMessage, err := gitclient.GetLatestCommitMessage(o.GitClient, o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to get last commit message")
	}
	lastCommitMessage = strings.TrimSpace(lastCommitMessage)
	log.Logger().Infof("found last commit message: %s", termcolor.ColorStatus(lastCommitMessage))

	if strings.Contains(lastCommitMessage, "/pipeline cancel") && !o.IsNewCluster {
		log.Logger().Infof("last commit disabled further processing")
		return nil
	}

	if o.PullRequest {
		return o.pullRequest()
	}

	regen := true
	if strings.HasPrefix(lastCommitMessage, "Merge pull request") || strings.HasPrefix(lastCommitMessage, "Merge branch") {
		changedExternalSecret, err := o.CheckLastCommitChangedExternalSecret(o.GitClient, o.Dir)
		if err != nil {
			return errors.Wrapf(err, "failed to check if last commit changed external secret")
		}
		if changedExternalSecret {
			log.Logger().Infof("last commit changed an ExternalSecret so still performing a full regenerate")
		} else if o.IsNewCluster {
			log.Logger().Infof("applying to new cluster so performing a full regenerate")
		} else {
			log.Logger().Infof("last commit was a merge pull request without changing an ExternalSecret so not regenerating")
			regen = false
		}
	}

	if regen {
		_, err := o.Regenerate()
		if err != nil {
			return errors.Wrapf(err, "failed to regenerate")
		}

		c := &cmdrunner.Command{
			Dir:  o.Dir,
			Name: "make",
			Args: []string{"regen-phase-3", "NEW_CLUSTER=" + strconv.FormatBool(o.IsNewCluster)},
		}
		err = o.RunCommand(c)
		if err != nil {
			return errors.Wrapf(err, "failed to regenerate phase 3")
		}
	} else {
		c := &cmdrunner.Command{
			Dir:  o.Dir,
			Name: "make",
			Args: []string{"regen-none"},
		}
		err = o.RunCommand(c)
		if err != nil {
			return errors.Wrapf(err, "failed to run regen-none hook")
		}
	}
	return nil
}

// Regenerate regenerates the kubernetes resources
func (o *Options) Regenerate() (bool, error) {
	firstSha, err := gitclient.GetLatestCommitSha(o.GitClient, o.Dir)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get the last commit sha")
	}

	c := &cmdrunner.Command{
		Dir:  o.Dir,
		Name: "make",
		Args: []string{"regen-phase-1", "NEW_CLUSTER=" + strconv.FormatBool(o.IsNewCluster)},
	}
	err = o.RunCommand(c)
	if err != nil {
		return false, errors.Wrapf(err, "failed to regenerate phase 1")
	}

	secondSha, err := gitclient.GetLatestCommitSha(o.GitClient, o.Dir)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get the last commit sha")
	}

	lastCommitMessage, err := gitclient.GetLatestCommitMessage(o.GitClient, o.Dir)
	if err != nil {
		return false, errors.Wrapf(err, "failed to get last commit message")
	}
	lastCommitMessage = strings.TrimSpace(lastCommitMessage)
	log.Logger().Infof("found last commit message: %s", termcolor.ColorStatus(lastCommitMessage))

	if strings.Contains(lastCommitMessage, "/pipeline cancel") && secondSha == firstSha {
		log.Logger().Infof("no commits so skipping regen-phase-2")
		return false, nil
	}

	c = &cmdrunner.Command{
		Dir:  o.Dir,
		Name: "make",
		Args: []string{"regen-phase-2", "NEW_CLUSTER=" + strconv.FormatBool(o.IsNewCluster)},
	}
	err = o.RunCommand(c)
	if err != nil {
		return false, errors.Wrapf(err, "failed to regenerate phase 2")
	}
	return true, nil
}

// Run runs the command
func (o *Options) RunCommand(c *cmdrunner.Command) error {
	log.Logger().Info(info(c.CLI()))
	c.Out = os.Stdout
	c.Err = os.Stderr
	_, err := o.CommandRunner(c)
	return err
}

func (o *Options) pullRequest() error {
	c := &cmdrunner.Command{
		Dir:  o.Dir,
		Name: "make",
		Args: []string{"pr-regen"},
	}
	err := o.RunCommand(c)
	if err != nil {
		return errors.Wrapf(err, "failed to regen pr")
	}
	return nil
}

func (o *Options) CheckLastCommitChangedExternalSecret(gitter gitclient.Interface, dir string) (bool, error) {
	text, err := gitter.Command(dir, "log", "-m", "-1", "--name-only", "--pretty=format:")
	if err != nil {
		return false, errors.Wrapf(err, "failed to get file changes")
	}
	text = strings.TrimSpace(text)
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || !strings.HasSuffix(line, ".yaml") || !strings.HasPrefix(line, "config-root") {
			continue
		}

		u := unstructured.Unstructured{}
		path := filepath.Join(dir, line)
		err := yamls.LoadFile(path, &u)
		if err != nil {
			log.Logger().Warnf("failed to read YAML file %s", path)
			continue
		}
		kind := u.GetKind()
		if kind == "ExternalSecret" {
			log.Logger().Infof("last commit included an ExternalSecret at %s so lets regenerate", path)
			return true, nil
		}
		log.Logger().Debugf("ignoring kind %s in file %s", kind, path)
	}
	return false, nil
}

func (o *Options) isNewCluster() bool {
	client, err := kube.LazyCreateKubeClientWithMandatory(nil, true)
	if err != nil {
		log.Logger().Errorf("Failed to create k8s client. Assuming this is a neww cluster: %v", err)
		return true
	}
	// If label team=jx is not set on namespace jx the cluster is considered new, as in that the jx-boot job has not run
	ns, err := client.CoreV1().Namespaces().Get(context.TODO(), "jx", metav1.GetOptions{})
	if err != nil {
		log.Logger().Infof("Can't find namespace jx. Assuming this is a new cluster: %v", err)
		return true
	}
	team, ok := ns.GetLabels()["team"]
	if !ok {
		log.Logger().Infof("Label team not found on namespace jx. Assuming this is a new cluster.")
		return true
	}
	if team != "jx" {
		log.Logger().Infof("Label team not set to jx on namespace jx. Assuming this is a new cluster.")
		return true
	}
	return false
}
