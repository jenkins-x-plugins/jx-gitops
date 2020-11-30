package versionstreamer

import (
	"fmt"
	"path/filepath"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/versionstream"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// Options the options for the command
type Options struct {
	Dir                  string
	VersionStreamDir     string
	VersionStreamURL     string
	VersionStreamRef     string
	CommandRunner        cmdrunner.CommandRunner
	QuietCommandRunner   cmdrunner.CommandRunner
	Requirements         *jxcore.RequirementsConfig
	RequirementsFileName string
	Resolver             *versionstream.VersionResolver
}

func (o *Options) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory that contains the jx-requirements.yml")
	cmd.Flags().StringVarP(&o.VersionStreamDir, "version-stream-dir", "", "", "the directory for the version stream. Defaults to 'versionStream' in the current --dir")
	cmd.Flags().StringVarP(&o.VersionStreamURL, "version-stream-url", "", "", "the git clone URL of the version stream. If not specified it defaults to the value in the jx-requirements.yml")
	cmd.Flags().StringVarP(&o.VersionStreamRef, "version-stream-ref", "", "", "the git ref (branch, tag, revision) of the version stream to git clone. If not specified it defaults to the value in the jx-requirements.yml")
}

const (
	defaultVersionStreamURL = "https://github.com/jenkins-x/jxr-versions.git"
	defaultVersionStreamRef = "master"
)

// Validate validates the options and populates any missing values
func (o *Options) Validate() error {
	var err error
	if o.Requirements == nil {
		var requirementsResource *jxcore.Requirements
		requirementsResource, o.RequirementsFileName, err = jxcore.LoadRequirementsConfig(o.Dir, false)
		if err != nil {
			return errors.Wrapf(err, "failed to load jx-requirements.yml")
		}
		o.Requirements = &requirementsResource.Spec
	}
	if o.VersionStreamURL == "" {
		o.VersionStreamURL = defaultVersionStreamURL
	}
	if o.VersionStreamRef == "" {
		o.VersionStreamRef = defaultVersionStreamRef

	}
	if o.VersionStreamDir == "" {
		o.VersionStreamDir = filepath.Join(o.Dir, "versionStream")
	}
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
	}
	if o.QuietCommandRunner == nil {
		o.QuietCommandRunner = cmdrunner.QuietCommandRunner
	}
	err = o.ResolveVersionStream()
	if err != nil {
		return errors.Wrapf(err, "failed to resolve the version stream")
	}
	if o.Resolver == nil {
		o.Resolver = &versionstream.VersionResolver{
			VersionsDir: o.VersionStreamDir,
		}
	}
	return nil
}

// ResolveVersionStream verifies there is a valid version stream and if not resolves it via kpt
func (o *Options) ResolveVersionStream() error {
	chartsDir := filepath.Join(o.VersionStreamDir, "charts")
	exists, err := files.DirExists(chartsDir)
	if err != nil {
		return errors.Wrapf(err, "failed to check version stream dir exists %s", chartsDir)
	}
	if exists {
		return nil
	}
	versionStreamPath, err := filepath.Rel(o.Dir, o.VersionStreamDir)
	if err != nil {
		return errors.Wrapf(err, "failed to get relative path of version stream %s in %s", o.VersionStreamDir, o.Dir)
	}

	// lets use kpt to copy the values file from the version stream locally
	c := &cmdrunner.Command{
		Dir:  o.Dir,
		Name: "kpt",
		Args: []string{
			"pkg",
			"get",
			fmt.Sprintf("%s/@%s", o.VersionStreamURL, o.VersionStreamRef),
			versionStreamPath,
		},
	}
	_, err = o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to resolve version stream %s ref %s using kpt", o.VersionStreamURL, o.VersionStreamRef)
	}
	return nil
}
