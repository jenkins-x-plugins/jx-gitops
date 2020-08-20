package update

import (
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/resolve"
	kptupdate "github.com/jenkins-x/jx-gitops/pkg/cmd/kpt/update"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// ShowOptions the options for viewing running PRs
type Options struct {
	kptupdate.Options
	HelmfileResolve resolve.Options
}

// NewCmdUpdate creates a command object
func NewCmdUpdate() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:   "update",
		Short: "Updates the git repository from the version stream",
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.Options.AddFlags(cmd)
	o.HelmfileResolve.AddFlags(cmd, "")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	log.Logger().Infof("upgrading local source code from the version stream using kpt...\n\n")

	err := o.Options.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to update source using kpt")
	}

	log.Logger().Infof("\nnow checking the chart versions in %s\n\n", termcolor.ColorInfo("helmfile.yaml"))

	o.HelmfileResolve.UpdateMode = true
	err = o.HelmfileResolve.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to update the helmfile versions")
	}
	return nil
}
