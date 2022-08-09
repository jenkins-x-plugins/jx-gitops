package template

import (
	"fmt"
	"os"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/helmfiles"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/plugins"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yaml2s"
	"github.com/roboll/helmfile/pkg/state"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const (
	useHelmfileRepos = false
)

var (
	cmdLong = templates.LongDesc(`
		Template the helmfile.yaml
`)

	cmdExample = templates.Examples(`
		# template the helmfile.yaml
		%s helmfile template
	`)
)

// Options the options for the command
type Options struct {
	options.BaseOptions
	Helmfile          string
	Helmfiles         []helmfiles.Helmfile
	KptBinary         string
	HelmfileBinary    string
	HelmBinary        string
	BatchMode         bool
	CommandRunner     cmdrunner.CommandRunner
	Sequencial        bool
	Dir               string
	IncludeCRDs       bool
	OutputDirTemplate string
	Concurrency       string
	TestOutOfCluster  bool
	Results           Results
}

type Results struct {
	RequirementsValuesFileName string
}

// NewCmdHelmfileTemplate creates a command object for the command
func NewCmdHelmfileTemplate() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "template",
		Short:   "Parallel template execution",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	o.BaseOptions.AddBaseFlags(cmd)

	if useHelmfileRepos {
		cmd.Flags().StringVarP(&o.HelmfileBinary, "helmfile-binary", "", "", "specifies the helmfile binary location to use. If not specified defaults to using the downloaded helmfile plugin")
	}
	cmd.Flags().StringVarP(&o.HelmBinary, "helm-binary", "", "", "specifies the helm binary location to use. If not specified defaults to using the downloaded helm plugin")
	o.AddFlags(cmd, "")
	return cmd, o
}

func (o *Options) AddFlags(cmd *cobra.Command, prefix string) {
	cmd.Flags().StringVarP(&o.OutputDirTemplate, "output-dir-template", "", "/tmp/generate/{{.Release.Namespace}}/{{.Release.Name}}", "")
	cmd.Flags().BoolVarP(&o.IncludeCRDs, "include-crds", "", true, "if CRDs should be included in the output")
	cmd.Flags().BoolVarP(&o.Sequencial, "sequential", "", true, "if run command sequentially")
	cmd.Flags().StringVarP(&o.Helmfile, "helmfile", "", "", "the helmfile to resolve. If not specified defaults to 'helmfile.yaml' in the dir")
	cmd.Flags().StringVarP(&o.Concurrency, "concurrency", "", "", "the helmfile to resolve. If not specified defaults to 'helmfile.yaml' in the dir")

}

// Validate validates the options and populates any missing values
func (o *Options) Validate() error {
	err := o.BaseOptions.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to ")
	}

	if o.Helmfile == "" {
		o.Helmfile = "helmfile.yaml"
	}

	if o.HelmfileBinary == "" {
		o.HelmfileBinary, err = plugins.GetHelmfileBinary(plugins.HelmfileVersion)
		if err != nil {
			return errors.Wrapf(err, "failed to download helmfile plugin")
		}
	}
	if o.HelmBinary == "" {
		o.HelmBinary, err = plugins.GetHelmBinary(plugins.HelmVersion)
		if err != nil {
			return errors.Wrapf(err, "failed to download helm plugin")
		}
	}

	if o.Dir == "" {
		o.Dir = "."
	}
	helmfiles, err := helmfiles.GatherHelmfiles(o.Helmfile, o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to gather nested helmfiles")
	}
	o.Helmfiles = helmfiles

	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
	}
	return nil
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to ")
	}

	if o.Sequencial {
		o.runCommand(o.Helmfile)
	}

	helmfiles, err := helmfiles.GatherHelmfiles(o.Helmfile, o.Dir)
	if err != nil {
		return errors.Wrapf(err, "error gathering helmfiles")
	}

	for _, helmfile := range helmfiles {
		err := o.processHelmfile(helmfile)
		if err != nil {
			return errors.Wrapf(err, "failed to process helmfile %s", helmfile.Filepath)
		}
		// ToDo: What are we trying to do here?

	}

	return nil
}

func (o *Options) processHelmfile(helmfile helmfiles.Helmfile) error {
	helmState := state.HelmState{}
	path := helmfile.Filepath
	err := yaml2s.LoadFile(path, &helmState)
	if err != nil {
		return errors.Wrapf(err, "failed to load helmfile %s", helmfile)
	}

	return nil
}

func (o *Options) runCommand(helmfile string) error {
	args := []string{}
	if o.HelmBinary != "" {
		args = append(args, "--helm-binary", o.HelmBinary)
	}
	if helmfile != "" {
		args = append(args, "--file", helmfile)
	}
	args = append(args, "template")
	// args = append(args, "--validate")
	if o.IncludeCRDs {
		args = append(args, "--include-crds")
	}
	if o.OutputDirTemplate != "" {
		args = append(args, "--output-dir-template", o.OutputDirTemplate)
	}
	if o.Concurrency != "" {
		args = append(args, "--concurrency", o.Concurrency)
	}

	c := &cmdrunner.Command{
		Dir:  o.Dir,
		Name: o.HelmfileBinary,
		Args: args,
		Out:  os.Stdout,
		Err:  os.Stderr,
	}
	_, err := o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to run command %s in dir %s", c.CLI(), o.Dir)
	}
	return nil
}
