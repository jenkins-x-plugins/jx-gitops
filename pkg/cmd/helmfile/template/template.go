package template

import (
	"context"
	"fmt"

	"github.com/cenkalti/backoff"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/helmfiles"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/plugins"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"

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
	ValidateRelease   bool
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
	cmd.Flags().BoolVarP(&o.Sequencial, "sequential", "", false, "if run command sequentially")
	cmd.Flags().BoolVarP(&o.ValidateRelease, "validate", "", false, "if run command sequentially")
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
		o.CommandRunner = commandRunner
	}
	return nil
}

// Run implements the command
func (o *Options) Run() error {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to ")
	}

	if o.Sequencial {
		log.Logger().Infof(termcolor.ColorStatus("------- sequential -----------"))
		command := o.buildCommand(o.Helmfile)
		result, err := o.CommandRunner(command)
		if err != nil {
			return errors.Wrapf(err, "failed to run command")
		}
		if result != "" {
			log.Logger().Infof(termcolor.ColorStatus(result))
		}
		return nil
	}
	log.Logger().Infof(termcolor.ColorStatus("------- parrallel -----------"))

	commands := []*cmdrunner.Command{}

	for _, helmfile := range o.Helmfiles[1:] {
		commands = append(commands, o.buildCommand(helmfile.Filepath))

	}
	cr := NewCommandRunners(10)
	cr.CommandRunner = o.CommandRunner
	go cr.GenerateFrom(commands)

	go cr.Run(ctx)

	for {
		select {
		case r, ok := <-cr.Results():
			if !ok {
				continue
			}
			log.Logger().Infof(termcolor.ColorStatus(r.Attempts))

			if r.Value != "" {
				log.Logger().Infof(termcolor.ColorStatus(r.Value))
			}
			if r.Err != nil {
				return errors.Wrapf(r.Err, "failed to run command")
				// log.Logger().Infof(termcolor.ColorStatus(r.Err))

			}

		case <-cr.Done:
			return nil
		}
	}

}

func (o *Options) buildCommand(helmfile string) *cmdrunner.Command {
	args := []string{}
	if o.HelmBinary != "" {
		args = append(args, "--helm-binary", o.HelmBinary)
	}
	if helmfile != "" {
		args = append(args, "--file", helmfile)
	}
	args = append(args, "template")
	if o.IncludeCRDs {
		args = append(args, "--include-crds")
	}
	if o.OutputDirTemplate != "" {
		args = append(args, "--output-dir-template", o.OutputDirTemplate)
	}
	if o.Concurrency != "" {
		args = append(args, "--concurrency", o.Concurrency)
	}
	if o.ValidateRelease {
		args = append(args, "--validate")
	}

	c := &cmdrunner.Command{
		Dir:                o.Dir,
		Name:               o.HelmfileBinary,
		Args:               args,
		ExponentialBackOff: backoff.NewExponentialBackOff(),
	}

	return c
}
func commandRunner(c *cmdrunner.Command) (string, error) {
	if c.Dir == "" {
		log.Logger().Infof("about to run: %s", termcolor.ColorInfo(cmdrunner.CLI(c)))
	} else {
		log.Logger().Infof("about to run: %s in dir %s", termcolor.ColorInfo(cmdrunner.CLI(c)), termcolor.ColorInfo(c.Dir))
	}
	result, err := c.Run()

	return result, err
}
