package template

import (
	"fmt"
	"sync"
	"time"

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
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to ")
	}

	if o.Sequencial {
		log.Logger().Infof(termcolor.ColorStatus("------- sequential -----------"))
		result, err := o.runCommand(o.Helmfile)
		if err != nil {
			return errors.Wrapf(err, "failed to run command")
		}
		if result != "" {
			log.Logger().Infof(termcolor.ColorStatus(result))
		}
		return nil
	}
	log.Logger().Infof(termcolor.ColorStatus("------- parrallel -----------"))

	// MD5All closes the done channel when it returns; it may do so before
	// receiving all the values from c and errc.
	done := make(chan struct{})
	defer close(done)
	errc := make(chan error, 1)

	helmfilesc := o.getHelmFiles(done)

	// Start a fixed number of goroutines to read and digest files.
	c := make(chan string) // HLc
	var wg sync.WaitGroup
	const numDigesters = 5
	wg.Add(numDigesters)
	for i := 0; i < numDigesters; i++ {
		go func() {
			o.digester(done, helmfilesc, c, errc) // HLc
			wg.Done()
			fmt.Println("******  returned *********")
		}()
	}
	go func() {
		wg.Wait()
		close(c) // HLc
	}()
	// End of pipeline. OMIT

	go func(c <-chan string) {

		select {
		case r := <-c: // HL
			log.Logger().Infof(termcolor.ColorStatus(r))
		case <-done: // HL
			return
		}

	}(c)

	// Check whether the Walk failed.

	if err := <-errc; err != nil { // HLerrc
		return err
	}

	return nil
}

func (o *Options) getHelmFiles(done <-chan struct{}) <-chan helmfiles.Helmfile {
	paths := make(chan helmfiles.Helmfile)

	go func() { // HL

		// Close the paths channel after Walk returns.
		defer close(paths) // HL
		// No select needed for this send, since errc is buffered.

		// Start sending jobs to the thread channel
		for _, helmfile := range o.Helmfiles {
			func(helmfile helmfiles.Helmfile) {

				select {
				case paths <- helmfile: // HL
				case <-done: // HL
					fmt.Println("--------- exiting get helmflies -----------")
					return
				}
			}(helmfile)

		}
	}()
	return paths

}

// digester reads path names from paths and sends digests of the corresponding
// files on c until either paths or done is closed.
func (o *Options) digester(done <-chan struct{}, paths <-chan helmfiles.Helmfile, c chan<- string, errc chan<- error) {
	for path := range paths { // HLpaths
		// result, err := o.runCommand(path.Filepath)
		result := path.Filepath
		// err := nil
		// if err != nil {
		// 	errc <- err
		// 	return
		// }
		select {
		case c <- result:
		case <-done:
			return
		}
	}
}

func (o *Options) runCommand(helmfile string) (string, error) {
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
		Dir:     o.Dir,
		Name:    o.HelmfileBinary,
		Args:    args,
		Timeout: 10 * time.Second,
	}
	result, err := commandRunner(c)
	if err != nil {
		return "", errors.Wrapf(err, "failed to run command %s in dir %s", c.CLI(), o.Dir)
	}

	return result, nil
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
