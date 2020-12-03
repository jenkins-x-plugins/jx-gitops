package template

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helmfile/move"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/rename"
	split2 "github.com/jenkins-x/jx-gitops/pkg/cmd/split"
	"github.com/jenkins-x/jx-gitops/pkg/helmhelpers"
	"github.com/jenkins-x/jx-gitops/pkg/jxtmpl/reqvalues"
	"github.com/jenkins-x/jx-gitops/pkg/plugins"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yaml2s"
	"github.com/jenkins-x/jx-kube-client/v3/pkg/kubeclient"
	"github.com/roboll/helmfile/pkg/state"

	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
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

	err = helmhelpers.AddHelmRepositories(o.HelmBinary, helmState, o.CommandRunner, nil)
	if err != nil {
		return errors.Wrapf(err, "failed to add helm repositories")
	}

	log.Logger().Infof("generating helm templates to dir %s", o.OutputDir)

	// if we only have one namespace we don't need to create a new file
	if len(namespaces) == 1 {
		log.Logger().Infof("only a single namespace used in the releases")

		for ns := range namespaces {
			return o.runHelmfile(o.Helmfile, ns, o.Args, &helmState)
		}
	}

	requirementsResource, _, err := jxcore.LoadRequirementsConfig(o.Dir, false)
	if err != nil {
		return errors.Wrapf(err, "failed to load jx-requirements.yml")
	}
	requirements := &requirementsResource.Spec

	globalEnvs := helmState.Environments["default"]
	var globalList []interface{}
	if len(globalEnvs.Values) > 0 {
		for _, v := range globalEnvs.Values {
			if v != "jx-values.yaml" {
				globalList = append(globalList, v)
			}
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

		jxValuesFile, err := o.createNamespaceJXValuesFile(requirements, ns)
		if err != nil {
			return errors.Wrapf(err, "failed to create jx-values.yaml file for namespace %s", ns)
		}

		// lets add the namespace specific jx-values.yaml file into the helmfile
		if helmState2.Environments == nil {
			helmState2.Environments = map[string]state.EnvironmentSpec{}
		}
		envs := helmState2.Environments["default"]
		envs.Values = append(globalList, jxValuesFile)
		helmState2.Environments["default"] = envs

		err = yaml2s.SaveFile(&helmState2, fileName)
		if err != nil {
			return errors.Wrapf(err, "failed to save helmfile %s", fileName)
		}

		args := strings.Replace(o.Args, "--values=jx-values.yaml", "--values=jx-values-"+ns+".yaml", 1)
		err = o.runHelmfile(fileName, ns, args, &helmState2)
		if err != nil {
			return errors.Wrapf(err, "failed to run helmfile template")
		}

		defer os.Remove(jxValuesFile)
		defer os.Remove(fileName)
	}
	return nil
}

func (o *Options) runHelmfile(fileName string, ns, helmfileArgs string, state *state.HelmState) error {
	outDir := filepath.Join(o.TmpDir, ns)

	err := os.MkdirAll(outDir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create directory %s", outDir)
	}

	args := []string{"--file", fileName}
	if o.Debug {
		args = append(args, "--debug")
	}
	args = append(args, "--namespace", ns, "template", "--include-crds")
	if helmfileArgs != "" {
		args = append(args, "-args", helmfileArgs)
	}
	args = append(args, "--output-dir", outDir)

	c := &cmdrunner.Command{
		Name: "helmfile",
		Args: args,
	}
	err = helmhelpers.RunCommandAndLogOutput(o.CommandRunner, c, debugInfoPrefixes, debugInfoSuffixes)
	if err != nil {
		return errors.Wrapf(err, "failed to run helmfile template")
	}

	// lets split any generated files into one file per resource...
	so := split2.Options{
		Dir: outDir,
	}
	err = so.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to split the generated helm resources in dir %s", outDir)
	}

	// now lets rename to canonical file names
	_, rn := rename.NewCmdRename()
	rn.Dir = outDir
	err = rn.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to rename helm output files in dir %s", outDir)
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

// createNamespaceJXValuesFile lets create a jx-values-$ns.yaml file for the namespace specific ingress changes
func (o *Options) createNamespaceJXValuesFile(requirements *jxcore.RequirementsConfig, ns string) (string, error) {
	req2 := *requirements
	defaultNS := jxcore.DefaultNamespace

	req2.Ingress.NamespaceSubDomain = strings.Replace(req2.Ingress.NamespaceSubDomain, defaultNS, ns, 1)

	// if we are in an environment with custom ingress lets use that
	for _, env := range requirements.Environments {
		if defaultNS+"-"+env.Key == ns && env.Ingress != nil {
			if env.Ingress.Domain != "" {
				req2.Ingress.Domain = env.Ingress.Domain
			}
			if env.Ingress.NamespaceSubDomain != "" {
				req2.Ingress.NamespaceSubDomain = env.Ingress.NamespaceSubDomain
			}
		}
	}

	fileName := filepath.Join(o.Dir, fmt.Sprintf("jx-values-%s.yaml", ns))
	err := reqvalues.SaveRequirementsValuesFile(&req2, fileName)
	if err != nil {
		return fileName, errors.Wrapf(err, "failed to save %s", fileName)
	}
	return fileName, nil
}
