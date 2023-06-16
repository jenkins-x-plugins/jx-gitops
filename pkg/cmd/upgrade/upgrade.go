package upgrade

import (
	"os"
	"path/filepath"
	"strings"

	markdown "github.com/MichaelMure/go-term-markdown"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helmfile/resolve"
	kptupdate "github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/kpt/update"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/plugins"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/tfupgrade"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/term"
)

// ShowOptions the options for viewing running PRs
type Options struct {
	kptupdate.Options
	HelmfileResolve  resolve.Options
	TerraformUpgrade tfupgrade.Options
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

	exists, err := o.HelmfileResolve.HasHelmfile()
	if err != nil {
		return errors.Wrapf(err, "failed to check for helmfile")
	}
	if exists {
		err = o.doHelmfileUpgrade()
		if err != nil {
			return errors.Wrapf(err, "failed to resolve helmfile")
		}
	}

	err = o.doTerraformUpgrade()
	if err != nil {
		return errors.Wrapf(err, "failed to upgrade terraform configuration")
	}

	o.DisplayReleaseNotes()
	return nil
}

func (o *Options) doHelmfileUpgrade() error {
	log.Logger().Infof("\nnow checking the chart versions in %s\n\n", termcolor.ColorInfo("helmfile.yaml"))
	var err error
	if o.HelmfileResolve.HelmBinary == "" {
		o.HelmfileResolve.HelmBinary, err = plugins.GetHelmBinary(plugins.HelmVersion)
		if err != nil {
			return errors.Wrapf(err, "failed to download helm binary")
		}
		log.Logger().Infof("using helm binary %s to verify chart repositories", termcolor.ColorInfo(o.HelmfileResolve.HelmBinary))
	}

	o.HelmfileResolve.UpdateMode = true
	err = o.HelmfileResolve.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to update the helmfile versions")
	}
	return nil
}

func (o *Options) doTerraformUpgrade() error {
	if o.Options.Dir != "" {
		o.TerraformUpgrade.Dir = o.Options.Dir
	}
	if o.HelmfileResolve.VersionStreamDir != "" {
		o.TerraformUpgrade.VersionStreamDir = o.HelmfileResolve.VersionStreamDir
	}
	err := o.TerraformUpgrade.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to upgrade terraform git repository versions")
	}
	return nil
}

// DisplayReleaseNotes Display untracked (new) files in notes directory
func (o *Options) DisplayReleaseNotes() {
	newNotes, err := o.GitClient.Command(o.Dir, "ls-files", "--exclude-standard", "--others", "versionStream/release-notes")
	if err != nil {
		log.Logger().Warnf("failed to find release notes: %s", err)
		return
	}
	if newNotes != "" {
		noteFiles := strings.Split(newNotes, "\n")
		for _, noteFile := range noteFiles {
			// Notes files in markdown and text format are supported. If suffix isn't .md text format is assumed.
			fileContent, err := os.ReadFile(filepath.Join(o.Dir, noteFile))
			if err != nil {
				log.Logger().Warnf("failed to load release notes file %s: %s", noteFile, err)
				continue
			}
			if strings.HasSuffix(noteFile, ".md") {
				width, _, err := term.GetSize(int(os.Stdout.Fd()))
				if err != nil || width > 115 {
					width = 115
				}
				fileContent = markdown.Render(string(fileContent), width, 0)
			}
			log.Logger().Infof("\n%s\n", fileContent)
		}
	}
}
