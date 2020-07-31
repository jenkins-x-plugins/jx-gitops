package versionstreamer

import (
	"io/ioutil"

	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/pkg/versionstream/versionstreamrepo"

	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-helpers/pkg/versionstream"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Options the options for the command
type Options struct {
	Dir                  string
	VersionStreamDir     string
	VersionStreamURL     string
	VersionStreamRef     string
	IOFileHandles        *files.IOFileHandles
	Gitter               gitclient.Interface
	CommandRunner        cmdrunner.CommandRunner
	Requirements         *config.RequirementsConfig
	RequirementsFileName string
	Resolver             *versionstream.VersionResolver
}

func (o *Options) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory that contains the jx-requirements.yml")
	cmd.Flags().StringVarP(&o.VersionStreamDir, "version-stream-dir", "", "", "optional directory that contains a version stream")
	cmd.Flags().StringVarP(&o.VersionStreamURL, "url", "n", "", "the git clone URL of the version stream. If not specified it defaults to the value in the jx-requirements.yml")
	cmd.Flags().StringVarP(&o.VersionStreamRef, "ref", "c", "", "the git ref (branch, tag, revision) of the version stream to git clone. If not specified it defaults to the value in the jx-requirements.yml")
}

// Validate validates the options and populates any missing values
func (o *Options) Validate() error {
	var err error
	if o.Requirements == nil {
		o.Requirements, o.RequirementsFileName, err = config.LoadRequirementsConfig(o.Dir, false)
		if err != nil {
			return errors.Wrapf(err, "failed to load jx-requirements.yml")
		}
	}
	requirements := o.Requirements
	if o.VersionStreamURL == "" {
		o.VersionStreamURL = requirements.VersionStream.URL
		if o.VersionStreamURL == "" {
			o.VersionStreamURL = requirements.VersionStream.URL
		}
	}
	if o.VersionStreamRef == "" {
		o.VersionStreamRef = requirements.VersionStream.Ref
		if o.VersionStreamRef == "" {
			o.VersionStreamRef = "master"
		}
	}
	if o.VersionStreamDir == "" {
		if o.VersionStreamURL == "" {
			return errors.Errorf("Missing option:  --%s ", termcolor.ColorInfo("url"))
		}

		var err error
		tmpDir, err := ioutil.TempDir("", "jx-version-stream-")
		if err != nil {
			return errors.Wrap(err, "failed to create temp dir")
		}

		o.VersionStreamDir, _, err = versionstreamrepo.CloneJXVersionsRepoToDir(tmpDir, o.VersionStreamURL, o.VersionStreamRef, nil, o.Git(), true, false, files.GetIOFileHandles(o.IOFileHandles))
		if err != nil {
			return errors.Wrapf(err, "failed to clone version stream to %s", o.Dir)
		}
	}

	if o.Resolver == nil {
		o.Resolver = &versionstream.VersionResolver{
			VersionsDir: o.VersionStreamDir,
		}
	}
	return nil
}

// Git returns the gitter - lazily creating one if required
func (o *Options) Git() gitclient.Interface {
	if o.Gitter == nil {
		o.Gitter = cli.NewCLIClient("", o.CommandRunner)
	}
	return o.Gitter
}
