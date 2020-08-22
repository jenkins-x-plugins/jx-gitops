package template

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/move"
	split2 "github.com/jenkins-x/jx-gitops/pkg/cmd/split"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/yaml2s"
	"github.com/jenkins-x/jx-kube-client/pkg/kubeclient"
	"github.com/roboll/helmfile/pkg/state"

	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-logging/pkg/log"
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
)

// Options the options for the command
type Options struct {
	Dir           string
	Helmfile      string
	Args          string
	OutputDir     string
	TmpDir        string
	Namespace     string
	Debug         bool
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

	return cmd, o
}

// Validate validates the options and populates any missing values
func (o *Options) Validate() error {
	if o.Helmfile == "" {
		o.Helmfile = filepath.Join(o.Dir, "helmfile.yaml")
	}
	var err error
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
		o.CommandRunner = cmdrunner.DefaultCommandRunner
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

	helmState := state.HelmState{}
	err = yaml2s.LoadFile(o.Helmfile, &helmState)
	if err != nil {
		return errors.Wrapf(err, "failed to load helmfile %s", o.Helmfile)
	}

	if o.Namespace == "" {
		o.Namespace, err = kubeclient.CurrentNamespace()
		if err != nil {
			return errors.Wrap(err, "failed to detect the current kubernetes namespace")
		}
	}

	if len(helmState.Releases) == 0 {
		return nil
	}

	namespaces := map[string]bool{}

	for i := range helmState.Releases {
		release := &helmState.Releases[i]
		if release.Namespace == "" {
			release.Namespace = o.Namespace
		}
		namespaces[release.Namespace] = true
	}

	if len(namespaces) == 0 {
		log.Logger().Warnf("no releases in file %s", o.Helmfile)
		return nil
	}

	for _, repo := range helmState.Repositories {
        	c := &cmdrunner.Command{
			Name: "helm",
			Args: []string{"repo", "add", repo.Name, repo.URL},
        	}
        	_, err = o.CommandRunner(c)	
	        if err != nil {
			return errors.Wrap(err, "failed to add helm repo")
		}

	        log.Logger().Infof("added helm repository %s %s", repo.Name, repo.URL)
	}

	log.Logger().Infof("generating helm templates to dir %s", o.OutputDir)

	// if we only have one namespace we don't need to create a new file
	if len(namespaces) == 1 {
		log.Logger().Infof("only a single namespace used in the releases")

		for ns := range namespaces {
			return o.runHelmfile(o.Helmfile, ns, &helmState)
		}
	}

	// lets make a separate helmfile for each namespace and apply it
	for ns := range namespaces {
		fileName := filepath.Join(o.Dir, "helmfile-namespace-"+ns+".yaml")
		helmState2 := helmState
		helmState2.Releases = nil

		for _, release := range helmState.Releases {
			if release.Namespace == ns {
				helmState2.Releases = append(helmState2.Releases, release)
			}
		}
		err = yaml2s.SaveFile(&helmState2, fileName)
		if err != nil {
			return errors.Wrapf(err, "failed to save helmfile %s", fileName)
		}

		err = o.runHelmfile(fileName, ns, &helmState2)
		if err != nil {
			return errors.Wrapf(err, "failed to run helmfile template")
		}

		defer os.Remove(fileName)
	}
	return nil
}

func (o *Options) runHelmfile(fileName string, ns string, state *state.HelmState) error {
	outDir := filepath.Join(o.TmpDir, ns)

	err := os.MkdirAll(outDir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create directory %s", outDir)
	}

	args := []string{"--file", fileName}
	if o.Debug {
		args = append(args, "--debug")
	}
	args = append(args, "--namespace", ns, "template")
	if o.Args != "" {
		args = append(args, "-args", o.Args)
	}
	args = append(args, "--output-dir", outDir)

	c := &cmdrunner.Command{
		Name: "helmfile",
		Args: args,
		Out:  os.Stdout,
		Err:  os.Stderr,
	}
	_, err = o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to run %s", c.CLI())
	}

	// lets split any generated files into one file per resource...
	so := split2.Options{
		Dir: outDir,
	}
	err = so.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to split the generated helm resources in dir %s", outDir)
	}

	// now lets move the generated resources to the real output dir
	mv := move.Options{
		Dir:             outDir,
		OutputDir:       o.OutputDir,
		SingleNamespace: ns,
		HelmState:       state,
	}
	err = mv.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to move the generated resources from temp dir %s", outDir)
	}
	return nil
}
