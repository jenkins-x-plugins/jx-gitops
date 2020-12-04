package template

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/move"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/rename"
	split2 "github.com/jenkins-x/jx-gitops/pkg/cmd/split"
	"github.com/jenkins-x/jx-gitops/pkg/helmhelpers"
	"github.com/jenkins-x/jx-gitops/pkg/plugins"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/roboll/helmfile/pkg/state"

	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Runs 'helmfile template' on the helmfile for each namespace putting the results in a separate folder
`)

	cmdExample = templates.Examples(`
		# splits the helmfile.yaml into separate files for each namespace and runs 'helm template' on each one	
		%s helmfile template --args="--include-crds --values=jx-values.yaml --values=src/fake-secrets.yaml.gotmpl" --output-dir config-root/namespaces
	`)

	// debugInfoPrefixes lets use debug level logging for lines starting with the following prefixes in the output of helmfile or helm commands
	debugInfoPrefixes = []string{
		"wrote ", "Templating ", "Adding repo ", "Fetching ", "Building dependency ",
	}

	// debugInfoSuffixes lets use debug level logging for lines ending with the following suffixes in the output of helmfile or helm commands
	debugInfoSuffixes = []string{
		" has been added to your repositories",
	}
)

// Options the options for the command
type Options struct {
	Dir           string
	Helmfile      string
	HelmBinary    string
	Args          string
	OutputDir     string
	TmpDir        string
	Namespace     string
	Debug         bool
	UseHelmPlugin bool
	CommandRunner cmdrunner.CommandRunner
}

type Results struct {
	HelmState                  state.HelmState
	RequirementsValuesFileName string
}

// NewCmdHelmfileTemplate creates a command object for the command
func NewCmdHelmfileTemplate() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "template",
		Short:   "Runs 'helmfile template' on the helmfile for each namespace putting the results in a separate folder",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}

	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to run the commands inside")
	cmd.Flags().StringVarP(&o.Helmfile, "helmfile", "", "", "the helmfile to template. Defaults to 'helmfile.yaml' in the directory")
	cmd.Flags().StringVarP(&o.Args, "args", "a", "", "the arguments passed through to helm")
	cmd.Flags().StringVarP(&o.OutputDir, "output-dir", "o", "", "the output directory. If not specified a temporary directory is created")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "n", "", "the default namespace if none is specified in the helmfile. Defaults to the current namespace")
	cmd.Flags().BoolVarP(&o.Debug, "debug", "", false, "enables debug logging in helmfile")
	cmd.Flags().BoolVarP(&o.UseHelmPlugin, "use-helm-plugin", "", false, "uses the jx binary plugin for helm rather than whatever helm is on the $PATH")

	return cmd, o
}

// Validate validates the options and populates any missing values
func (o *Options) Validate() error {
	var err error
	if o.Helmfile == "" {
		o.Helmfile = filepath.Join(o.Dir, "helmfile.yaml")
	}
	if o.HelmBinary == "" {
		if o.UseHelmPlugin {
			o.HelmBinary, err = plugins.GetHelmBinary(plugins.HelmVersion)
			if err != nil {
				return err
			}
		}
		if o.HelmBinary == "" {
			o.HelmBinary = "helm"
		}
	}
	if o.OutputDir == "" {
		o.OutputDir, err = ioutil.TempDir("", "")
		if err != nil {
			return errors.Wrapf(err, "failed to create temporary output directory")
		}
	}
	if o.TmpDir == "" {
		o.TmpDir, err = ioutil.TempDir("", "")
		if err != nil {
			return errors.Wrapf(err, "failed to create temporary work directory")
		}
	}
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.QuietCommandRunner
	}
	exists, err := files.FileExists(o.Helmfile)
	if err != nil {
		return errors.Wrapf(err, "failed to check if file exists %s", o.Helmfile)
	}
	if !exists {
		return errors.Errorf("helmfile %s does not exist", o.Helmfile)
	}
	return nil
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate")
	}

	err = o.templateHelmfile()

	if err != nil {
		return errors.Wrapf(err, "failed to run helmfile template")
	}

	err = o.structureTemplateOutput()

	if err != nil {
		return errors.Wrapf(err, "failed to restructure helmfile template output")
	}

	return nil
}

func (o Options) structureTemplateOutput() error {

	// lets split any generated files into one file per resource...
	so := split2.Options{
		Dir: o.TmpDir,
	}
	err := so.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to split the generated helm resources in dir %s", o.TmpDir)
	}

	// now lets rename to canonical file names
	_, rn := rename.NewCmdRename()
	rn.Dir = o.TmpDir
	err = rn.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to rename helm output files in dir %s", o.TmpDir)
	}

	// now lets move the generated resources to the real output dir
	mv := move.Options{
		Dir:       o.TmpDir,
		OutputDir: o.OutputDir,
	}
	err = mv.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to move the generated resources from temp dir %s", o.TmpDir)
	}
	return nil
}

func (o Options) templateHelmfile() error {

	args := []string{"--file", o.Helmfile, "template", "--include-crds", "--output-dir", o.TmpDir}
	if o.Args != "" {
		args = append(args, "--args", o.Args)
	}
	args = append(args, "--output-dir-template", "{{ .OutputDir }}/{{ .Release.Namespace }}")
	if o.Debug {
		args = append(args, "--debug")
	}
	c := &cmdrunner.Command{
		Name: "helmfile",
		Args: args,
	}
	err := helmhelpers.RunCommandAndLogOutput(o.CommandRunner, c, debugInfoPrefixes, debugInfoSuffixes)
	if err != nil {
		return errors.Wrapf(err, "failed to run helmfile template")
	}
	return nil
}
