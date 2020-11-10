package upgrade

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx-gitops/pkg/plugins"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/extensions"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/homedir"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdPluginsLong = templates.LongDesc(`
		Upgrades the binary plugins for this plugin
`)

	cmdPluginsExample = templates.Examples(`
		# upgrades your plugin binaries for gitops
		%s plugins upgrade
	`)
)

// UpgradeOptions the options for upgrading a cluster
type Options struct {
	CommandRunner cmdrunner.CommandRunner
	Path          string
}

// NewCmdUpgrade creates a command object for the command
func NewCmdUpgradePlugins() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "upgrade",
		Short:   "Upgrades the binary plugins for this plugin",
		Long:    cmdPluginsLong,
		Example: fmt.Sprintf(cmdPluginsExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Path, "path", "", "", "creates a symlink to the binary plugins in this bin path dir")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	pluginBinDir, err := homedir.DefaultPluginBinDir()
	if err != nil {
		return errors.Wrap(err, "failed to find plugin bin directory")
	}

	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
	}
	if o.Path != "" {
		err = os.MkdirAll(o.Path, files.DefaultDirWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to make bin directory %s", o.Path)
		}
	}

	for k := range plugins.Plugins {
		p := plugins.Plugins[k]
		log.Logger().Infof("checking binary jx plugin %s version %s is installed", termcolor.ColorInfo(p.Name), termcolor.ColorInfo(p.Spec.Version))
		fileName, err := extensions.EnsurePluginInstalled(p, pluginBinDir)
		if err != nil {
			return errors.Wrapf(err, "failed to ensure plugin is installed %s", p.Name)
		}

		if o.Path != "" {
			binName := filepath.Join(o.Path, p.Name)
			exists, err := files.FileExists(binName)
			if err != nil {
				return errors.Wrapf(err, "failed to check if file exists %s", binName)
			}
			if exists {
				err = os.Remove(binName)
				if err != nil {
					return errors.Wrapf(err, "failed to remove existing file %s", binName)
				}
			}
			err = os.Symlink(fileName, binName)
			if err != nil {
				return errors.Wrapf(err, "failed to create symlink from %s => %s", fileName, binName)
			}
			log.Logger().Infof("created symlink from %s => %s", fileName, binName)
		}
	}
	return nil
}
