package label

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Adds a label to the current pull request
`)

	cmdExample = templates.Examples(`
		# add label
		%s pr label -n mylabel 

		# add label if there exists a matching label with the regex
		%s pr label -n mylabel --matches "env/.*"
	`)
)

// Options the options for the command
type Options struct {
	options.BaseOptions
	scmhelpers.PullRequestOptions

	Label      string
	Regex      string
	Result     *scm.PullRequest
	LabelAdded bool
	re         *regexp.Regexp
}

// NewCmdPullRequestLabel creates a command object for the command
func NewCmdPullRequestLabel() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "label",
		Short:   "Add label to the pull request",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName, rootcmd.BinaryName),
		Run: func(_ *cobra.Command, _ []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.BaseOptions.AddBaseFlags(cmd)
	o.PullRequestOptions.AddFlags(cmd)

	cmd.Flags().StringVarP(&o.Label, "name", "n", "", "name of the label to add")
	cmd.Flags().StringVarP(&o.Regex, "matches", "m", "", "only label the Pull Request if there is already a label which matches the regular expression")
	cmd.Flags().BoolVarP(&o.IgnoreMissingPullRequest, "ignore-no-pr", "", false, "if an error is returned finding the Pull Request (maybe due to missing environment variables to find the PULL_NUMBER) just push to the current branch instead")
	return cmd, o
}

// Validate validates the command line options
func (o *Options) Validate() error {
	err := o.BaseOptions.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate base options")
	}

	err = o.PullRequestOptions.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate PR options ")
	}

	if o.Label == "" {
		return options.MissingOption("name")
	}
	if o.Regex != "" {
		var err error
		o.re, err = regexp.Compile(o.Regex)
		if err != nil {
			return errors.Wrapf(err, "failed to parse matches regular expression: %s", o.Regex)
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
	pr, err := o.DiscoverPullRequest()
	if err != nil {
		if o.IgnoreMissingPullRequest {
			log.Logger().Infof("could not find Pull Request so not labelling it")
			return nil
		}
		return errors.Wrapf(err, "failed to discover the pull request")
	}
	if pr == nil {
		return errors.Errorf("no Pull Request could be found for %d in repository %s", o.Number, o.Repository)
	}
	if len(pr.Labels) == 0 {
		// lets fetch the labels if git provider does not include them OOTB such as for things like BitBucketServer
		ctx := context.TODO()
		repo := pr.Repository()
		repoName := repo.FullName
		if repoName == "" {
			repoName = scm.Join(repo.Namespace, repo.Name)
		}
		pr.Labels, _, err = o.PullRequestOptions.ScmClient.PullRequests.ListLabels(ctx, repoName, pr.Number, &scm.ListOptions{})
		if err != nil {
			return errors.Wrapf(err, "failed to query Labels for repo %s and PullRequest %d", repoName, pr.Number)
		}
	}
	return o.labelPullRequest(pr)
}

func (o *Options) labelPullRequest(pr *scm.PullRequest) error {
	o.Result = pr
	label := o.Label

	matches := o.re == nil
	var labelNames []string
	for _, l := range pr.Labels {
		name := l.Name
		labelNames = append(labelNames, name)
		if name == label {
			log.Logger().Infof("pull request %s already has label %s", info(pr.Link), info(label))
			return nil
		}
		if o.re != nil {
			if o.re.MatchString(name) {
				log.Logger().Infof("pull request %s has matching label %s so lets label it...", info(pr.Link), info(name))
				matches = true
			}
		}
	}

	if !matches {
		sort.Strings(labelNames)
		log.Logger().Infof("pull request %s does not have a label matching %s. Has labels: %s", info(pr.Link), info(o.Regex), info(strings.Join(labelNames, " ")))
		return nil
	}

	ctx := context.Background()
	_, err := o.ScmClient.PullRequests.AddLabel(ctx, o.FullRepositoryName, o.Number, label)
	if err != nil {
		return errors.Wrapf(err, "failed to add label %s to pull request %s", label, pr.Link)
	}
	log.Logger().Infof("added label %s to pull request %s", info(label), info(pr.Link))

	o.LabelAdded = true
	return nil
}
