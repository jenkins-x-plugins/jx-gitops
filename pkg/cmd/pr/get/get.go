package get

import (
	"fmt"

	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"sigs.k8s.io/yaml"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Gets a pull request and displays fields from it
`)

	cmdExample = templates.Examples(`
		# display the head source URL
		%s pr get --head-url 
	`)
)

// Options the options for the command
type Options struct {
	scmhelpers.PullRequestOptions

	ShowHeadURL bool
	Result      *scm.PullRequest
}

// NewCmdPullRequestPush creates a command object for the command
func NewCmdPullRequestGet() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "get",
		Short:   "Gets a pull request and displays fields from it",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.PullRequestOptions.AddFlags(cmd)

	cmd.Flags().BoolVarP(&o.ShowHeadURL, "head-url", "", false, "show the head clone URL of the PR")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	err := o.PullRequestOptions.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to ")
	}
	pr, err := o.DiscoverPullRequest()
	if err != nil {
		return errors.Wrapf(err, "failed to discover the pull request")
	}
	if pr == nil {
		return errors.Errorf("no Pull Request could be found for %d in repository %s", o.Number, o.Repository)
	}
	return o.displayPullRequest(pr)
}

func (o *Options) displayPullRequest(pr *scm.PullRequest) error {
	o.Result = pr

	if o.ShowHeadURL {
		log.Logger().Info(pr.Head.Repo.Clone)
		return nil
	}

	data, err := yaml.Marshal(pr)
	if err != nil {
		return errors.Wrapf(err, "failed to marshal PullRequest as YAML")
	}
	log.Logger().Info(string(data))
	return nil

}
