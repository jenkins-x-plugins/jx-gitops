package versionstream

import (
	"fmt"
	"path/filepath"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"

	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"

	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"

	"sigs.k8s.io/kustomize/kyaml/yaml"

	"github.com/jenkins-x/jx-logging/v3/pkg/log"

	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"

	"github.com/jenkins-x/jx-helpers/v3/pkg/files"

	"github.com/pkg/errors"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/spf13/cobra"
)

var (
	createLong = templates.LongDesc(`
		Administer the cluster version stream settings
`)

	createExample = templates.Examples(`
		# switch to LTS (stable) version stream
		%s versionstream --lts

		# switch to latest version stream
		%s versionstream --latest

		# switch to a custom version stream
		%s versionstream --custom --url https://github.com/foo/bar.git --ref main
	`)
)

const (
	ltsVersionStreamURL    = "https://github.com/jenkins-x/jx3-lts-versions"
	latestVersionStreamURL = "https://github.com/jenkins-x/jxr-versions"
)

// Options the options for creating a repository
type Options struct {
	LTS         bool
	Latest      bool
	Custom      bool
	DoGitCommit bool
	GitURL      string
	GitRef      string
	GitDir      string
	Cmd         *cobra.Command
	Args        []string
	Dir         string
	Gitter      gitclient.Interface
}

// NewCmdVersionstream administer the cluster version stream
func NewCmdVersionstream() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "versionstream",
		Short:   "Administer the cluster version stream settings",
		Long:    createLong,
		Example: fmt.Sprintf(createExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			o.Cmd = cmd
			o.Args = args
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.Cmd = cmd

	cmd.Flags().BoolVarP(&o.LTS, "lts", "", false, "Switch the cluster version stream to the LTS (long term support on monthly release cadence) git repo, https://github.com/jenkins-x/jx3-lts-versions")
	cmd.Flags().BoolVarP(&o.Latest, "latest", "", false, "Switch the cluster version stream to the latest (latest releases) git repo, https://github.com/jenkins-x/jxr-versions")
	cmd.Flags().BoolVarP(&o.Custom, "custom", "", false, "Switch the cluster version stream to a custom version stream, requires url and ref flags set")
	cmd.Flags().StringVarP(&o.GitURL, "url", "", "", "The git URL to clone to fetch the initial set of files for a helm 3 / helmfile based git configuration if this command is not run inside a git clone or against a GitOps based cluster")
	cmd.Flags().StringVarP(&o.GitRef, "ref", "", "", "The kind of git server for the development environment")
	cmd.Flags().StringVarP(&o.GitDir, "directory", "", "/", "The directory used in the versionstream, defaults to root")

	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {

	err := o.switchVersionStream()
	if err != nil {
		return errors.Wrapf(err, "failed to switch versionstream")
	}

	err = o.GitCommit()
	if err != nil {
		return errors.Wrapf(err, "failed to commit updated Kptfile")
	}
	log.Logger().Infof("your version stream has been switched, please now run")
	log.Logger().Infof("%s", termcolor.ColorInfo("jx upgrade cli"))
	log.Logger().Infof("%s", termcolor.ColorInfo("jx gitops upgrade"))

	return nil
}

// Git returns the gitter - lazily creating one if required
func (o *Options) Git() gitclient.Interface {
	if o.Gitter == nil {
		o.Gitter = cli.NewCLIClient("", cmdrunner.DefaultCommandRunner)
	}
	return o.Gitter
}

func (o *Options) switchVersionStream() error {
	if !o.atLeastOneFlagSet() {
		return errors.New("select at least one flag, lts, latest or custom")
	}

	if o.moreThanOneFlagSet() {
		return errors.New("select only one flag, lts, latest or custom")
	}

	if o.Custom {
		if o.GitURL == "" || o.GitRef == "" {
			return errors.New("url and ref flags must be set if using custom")
		}
	}

	if o.Dir == "" {
		o.Dir = "."
	}
	kptFilePath := filepath.Join(o.Dir, "versionStream", "Kptfile")
	kptFileExist, err := files.FileExists(kptFilePath)
	if err != nil {
		return errors.Wrapf(err, "failed to check if %s exists", kptFilePath)
	}
	if !kptFileExist {
		return fmt.Errorf("file to find %s, clone you cluster git repository and rerun command", kptFilePath)
	}

	node, err := yaml.ReadFile(kptFilePath)
	if err != nil {
		return errors.Wrapf(err, "failed to load file %s", kptFilePath)
	}

	modified, err := o.modifyFn(node, kptFilePath)
	if err != nil {
		return errors.Wrapf(err, "failed to modify file %s", kptFilePath)
	}

	if !modified {
		return nil
	}

	err = yaml.WriteFile(node, kptFilePath)
	if err != nil {
		return errors.Wrapf(err, "failed to save %s", kptFilePath)
	}
	return nil
}

func (o *Options) modifyFn(node *yaml.RNode, path string) (bool, error) {

	repo := ""
	ref := ""
	directory := ""
	switch true {
	case o.LTS:
		repo = ltsVersionStreamURL
		ref = "main"
		directory = "versionStream"
	case o.Latest:
		repo = latestVersionStreamURL
		ref = "master"
		directory = "/"
	case o.Custom:
		repo = o.GitURL
		ref = o.GitRef
		directory = o.GitDir
	}

	err := node.PipeE(yaml.LookupCreate(yaml.ScalarNode, "upstream", "git", "repo"), yaml.FieldSetter{StringValue: repo})
	if err != nil {
		return false, errors.Wrapf(err, "failed to set the git source repository to %s for %s", repo, path)
	}
	err = node.PipeE(yaml.LookupCreate(yaml.ScalarNode, "upstream", "git", "ref"), yaml.FieldSetter{StringValue: ref})
	if err != nil {
		return false, errors.Wrapf(err, "failed to set the git source ref to %s for %s", ref, path)
	}
	err = node.PipeE(yaml.LookupCreate(yaml.ScalarNode, "upstream", "git", "directory"), yaml.FieldSetter{StringValue: directory})
	if err != nil {
		return false, errors.Wrapf(err, "failed to set the git directory to %s for %s", ref, path)
	}
	return true, nil
}

func (o *Options) moreThanOneFlagSet() bool {
	return ((o.LTS && o.Latest) || (o.Latest && o.Custom) || (o.LTS && o.Custom))
}

func (o *Options) atLeastOneFlagSet() bool {
	return o.LTS || o.Latest || o.Custom
}

func (o *Options) GitCommit() error {
	// git add / commit the kptfile change
	message := "chore: switch verionsstream to %s"
	switch true {
	case o.LTS:
		message = fmt.Sprintf(message, ltsVersionStreamURL)
	case o.Latest:
		message = fmt.Sprintf(message, latestVersionStreamURL)
	case o.Custom:
		message = fmt.Sprintf(message, o.GitURL)
	}

	gitter := o.Git()
	dir := "versionStream"
	_, err := gitclient.AddAndCommitFiles(gitter, dir, message)
	if err != nil {
		return errors.Wrapf(err, "failed to commit changes to git in dir %s", dir)
	}
	return nil
}
