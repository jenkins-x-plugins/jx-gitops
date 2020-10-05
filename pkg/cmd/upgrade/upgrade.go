package upgrade

import (
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/resolve"
	kptupdate "github.com/jenkins-x/jx-gitops/pkg/cmd/kpt/update"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// ShowOptions the options for viewing running PRs
type Options struct {
	kptupdate.Options
	HelmfileResolve resolve.Options
}

// NewCmdUpgrade creates a command object
func NewCmdUpgrade() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "upgrade",
		Aliases: []string{"update"},
		Short:   "Upgrades the GitOps git repository with the latest configuration and versions the Version Stream",
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
