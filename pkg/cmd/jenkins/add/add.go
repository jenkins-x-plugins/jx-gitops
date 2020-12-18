package add

import (
	"fmt"
	"path/filepath"
	"strings"

	helmfileadd "github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/add"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/naming"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Adds a new Jenkins server to the git repository
`)

	cmdExample = templates.Examples(`
		# adds a new jenkins server to the git repository
		%s jenkins add --name myjenkins

	`)
)

// Options the options for the command
type Options struct {
	helmfileadd.Options
	Name string
}

// NewCmdJenkinsAdd creates a command object for the command
func NewCmdJenkinsAdd() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "add",
		Aliases: []string{"create", "new"},
		Short:   "Adds a new Jenkins server to the git repository",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Name, "name", "n", "", "the name of the jenkins server to add")
	cmd.Flags().StringVarP(&o.Chart, "chart", "c", "jenkinsci/jenkins", "the jenkins helm chart to use")
	cmd.Flags().StringVarP(&o.Repository, "repository", "r", "https://charts.jenkins.io", "the helm chart repository URL of the chart")
	cmd.Flags().StringVarP(&o.Version, "version", "v", "", "the version of the helm chart. If not specified the versionStream will be checked otherwise the latest version is used")
	return cmd, o
}

func (o *Options) Run() error {
	o.Name = strings.TrimSpace(o.Name)
	if o.Name == "" {
		return options.MissingOption("name")
	}
	o.Name = naming.ToValidName(o.Name)
	o.Namespace = o.Name
	o.ReleaseName = "jenkins"
	o.Helmfile = filepath.Join(o.Dir, "helmfiles", o.Namespace, "helmfile.yaml")

	err := o.Options.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to add jenkins helm chart for %s", o.Name)
	}
	log.Logger().Infof("added helmfile %s for jenkins server %s", info(o.Helmfile), info(o.Name))

	// lets add the jenkins-resources chart too
	o.Chart = "jx3/jenkins-resources"
	o.ReleaseName = "jenkins-resources"
	o.Values = nil
	err = o.Options.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to add jenkins resources helm chart for %s", o.Name)
	}
	return nil
}
