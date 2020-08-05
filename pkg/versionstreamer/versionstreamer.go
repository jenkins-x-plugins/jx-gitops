package versionstreamer

import (
	"fmt"
	"path/filepath"

	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/files"
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
	CommandRunner        cmdrunner.CommandRunner
	Requirements         *config.RequirementsConfig
	RequirementsFileName string
	Resolver             *versionstream.VersionResolver
}

func (o *Options) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory that contains the jx-requirements.yml")
	cmd.Flags().StringVarP(&o.VersionStreamDir, "version-stream-dir", "", "", "the directory for the version stream. Defaults to 'versionStream' in the current --dir")
	cmd.Flags().StringVarP(&o.VersionStreamURL, "version-stream-url", "n", "", "the git clone URL of the version stream. If not specified it defaults to the value in the jx-requirements.yml")
	cmd.Flags().StringVarP(&o.VersionStreamRef, "version-stream-ref", "c", "", "the git ref (branch, tag, revision) of the version stream to git clone. If not specified it defaults to the value in the jx-requirements.yml")
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
		o.VersionStreamDir = filepath.Join(o.Dir, "versionStream")
	}
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
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
