package helm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/extsecret"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/split"
	"github.com/jenkins-x/jx-gitops/pkg/plugins"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/pkg/options"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	helmTemplateLong = templates.LongDesc(`
		Generate the kubernetes resources from a helm chart
`)

	helmTemplateExample = templates.Examples(`
		# generates the resources from a helm chart
		%s step helm template
	`)
)

// HelmTemplateOptions the options for the command
type TemplateOptions struct {
	OutDir           string
	HelmBinary       string
	ReleaseName      string
	Namespace        string
	Chart            string
	ValuesFiles      []string
	DefaultDomain    string
	GitCommitMessage string
	Version          string
	Repository       string
	BatchMode        bool
	NoGitCommit      bool
	NoSplit          bool
	NoExtSecrets     bool
	IncludeCRDs      bool
	CheckExists      bool
	Gitter           gitclient.Interface
	CommandRunner    cmdrunner.CommandRunner
}

// NewCmdHelmTemplate creates a command object for the command
func NewCmdHelmTemplate() (*cobra.Command, *TemplateOptions) {
	o := &TemplateOptions{}

	cmd := &cobra.Command{
		Use:     "template",
		Short:   "Generate the kubernetes resources from a helm chart",
		Long:    helmTemplateLong,
		Example: fmt.Sprintf(helmTemplateExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.OutDir, "output-dir", "o", "", "the output directory to generate the templates to. Defaults to charts/$name/resources")
	cmd.Flags().StringVarP(&o.ReleaseName, "name", "n", "", "the name of the helm release to template. Defaults to $APP_NAME if not specified")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", "", "specifies the namespace to use to generate the templates in")
	cmd.Flags().StringVarP(&o.Chart, "chart", "c", "", "the chart name to template. Defaults to 'charts/$name'")
	cmd.Flags().StringArrayVarP(&o.ValuesFiles, "values", "f", nil, "the helm values.yaml file used to template values in the generated template")
	cmd.Flags().StringVarP(&o.Version, "version", "v", "", "the version of the helm chart to use. If not specified then the latest one is used")
	cmd.Flags().StringVarP(&o.Repository, "repository", "r", "", "the helm chart repository to locate the chart")
	cmd.Flags().StringVarP(&o.GitCommitMessage, "commit-message", "", "chore: generated kubernetes resources from helm chart", "the git commit message used")

	o.AddFlags(cmd)
	return cmd, o
}

func (o *TemplateOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.DefaultDomain, "domain", "", "cluster.local", "the default domain name in the generated ingress")
	cmd.Flags().BoolVarP(&o.NoGitCommit, "no-git-commit", "", false, "if set then the command will not git add/commit the generated resources")
	cmd.Flags().BoolVarP(&o.NoSplit, "no-split", "", false, "if set then disable splitting of multiple resources into separate files")
	cmd.Flags().BoolVarP(&o.NoExtSecrets, "no-external-secrets", "", false, "if set then disable converting Secret resources to ExternalSecrets")
	cmd.Flags().BoolVarP(&o.IncludeCRDs, "include-crds", "", true, "if CRDs should be included in the output")
	cmd.Flags().BoolVarP(&o.CheckExists, "optional", "", false, "check if there is a charts dir and if not do nothing if it does not exist")
}

// Run implements the command
func (o *TemplateOptions) Run() error {
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
	}
	var err error
	bin := o.HelmBinary
	if bin == "" {
		bin, err = plugins.GetHelmBinary(plugins.HelmVersion)
		if err != nil {
			return err
		}
	}

	name := o.ReleaseName
	if name == "" {
		name = os.Getenv("APP_NAME")
		if name == "" {
			name = os.Getenv("REPO_NAME")
			if name == "" {
				return options.MissingOption("name")
			}
		}
	}
	chart := o.Chart
	if chart == "" {
		chart = filepath.Join("charts", name)
	}

	if o.Repository == "" {
		exists, err := files.DirExists(chart)
		if err != nil {
			return errors.Wrapf(err, "failed to check if dir exists %s", chart)
		}
		if !exists {
			if o.CheckExists {
				log.Logger().Infof("no charts dir so doing nothing %s", chart)
				return nil
			}
			return errors.Errorf("there is no chart at %s - you could try supply --chart", chart)
		}
	}
	outDir := o.OutDir
	if outDir == "" {
		outDir = filepath.Join(chart, "resources")
	}
	err = os.MkdirAll(outDir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure output directory exists %s", outDir)
	}

	tmpDir, err := ioutil.TempDir("", "")
	if err != nil {
		return errors.Wrap(err, "failed to create temporary directory")
	}

	tmpChartDir := ""
	if o.Repository != "" {
		tmpChartDir, err = ioutil.TempDir("", "")
		if err != nil {
			return errors.Wrap(err, "failed to create temporary chart directory")
		}

		// lets fetch the chart
		args := []string{"fetch", "--untar", "--repo", o.Repository}
		if o.Version != "" {
			args = append(args, "--version", o.Version)
		}
		args = append(args, name)

		c := &cmdrunner.Command{
			Name: bin,
			Args: args,
			Dir:  tmpChartDir,
			Out:  os.Stdout,
			Err:  os.Stderr,
		}
		log.Logger().Infof("about to run %s", termcolor.ColorInfo(c.CLI()))
		_, err = o.CommandRunner(c)
		if err != nil {
			return errors.Wrapf(err, "failed to run %s", c.CLI())
		}
	}

	cmdDir := ""

	args := []string{"template", "--output-dir", tmpDir}
	for _, valuesFile := range o.ValuesFiles {
		args = append(args, "--values", valuesFile)
	}

	if o.Repository != "" {
		args = append(args, "--repo", o.Repository)
		cmdDir = tmpChartDir
	}
	if o.Namespace != "" {
		args = append(args, "--namespace", o.Namespace)
	}
	if o.Version != "" {
		args = append(args, "--version", o.Version)
	}
	if o.IncludeCRDs {
		args = append(args, "--include-crds")
	}
	args = append(args, name, chart)
	c := &cmdrunner.Command{
		Name: bin,
		Args: args,
		Dir:  cmdDir,
		Out:  os.Stdout,
		Err:  os.Stderr,
	}
	log.Logger().Infof("about to run %s", termcolor.ColorInfo(c.CLI()))
	results, err := o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to run %s got: %s", c.CLI(), results)
	}

	// now lets copy the templates from the temp dir to the outDir
	crdsDir := filepath.Join(tmpDir, name, "crds")
	exists, err := files.DirExists(crdsDir)
	if err != nil {
		return errors.Wrapf(err, "failed to check if crds dir was generated")
	}
	if exists {
		err = util.CopyDirOverwrite(crdsDir, outDir)
		if err != nil {
			return errors.Wrapf(err, "failed to copy generated crds at %s to %s", crdsDir, outDir)
		}
	}
	templatesDir := filepath.Join(tmpDir, name, "templates")
	exists, err = files.DirExists(templatesDir)
	if err != nil {
		return errors.Wrapf(err, "failed to check if templates dir was generated")
	}
	if !exists {
		return errors.Errorf("no templates directory was created at %s", templatesDir)
	}
	err = files.CopyDirOverwrite(templatesDir, outDir)
	if err != nil {
		return errors.Wrapf(err, "failed to copy generated templates at %s to %s", templatesDir, outDir)
	}
	err = os.RemoveAll(tmpDir)
	if err != nil {
		return errors.Wrapf(err, "failed to remove tmp dir %s", tmpDir)
	}
	if !o.NoSplit {
		so := &split.Options{
			Dir: outDir,
		}
		err = so.Run()
		if err != nil {
			return errors.Wrapf(err, "failed to split YAML files at %s", outDir)
		}
	}
	if !o.NoExtSecrets {
		_, eo := extsecret.NewCmdExtSecrets()
		eo.Dir = outDir
		err = eo.Run()
		if err != nil {
			return errors.Wrapf(err, "failed to convert to external Secrets at %s", outDir)
		}
	}
	if o.NoGitCommit {
		return nil
	}
	return o.GitCommit(outDir, o.GitCommitMessage)
}

func (o *TemplateOptions) GitCommit(outDir string, commitMessage string) error {
	gitter := o.Git()
	_, err := gitter.Command(outDir, "add", "*")
	if err != nil {
		return errors.Wrapf(err, "failed to add generated resources to git in dir %s", outDir)
	}
	err = gitclient.CommitIfChanges(gitter, outDir, commitMessage)
	if err != nil {
		return errors.Wrapf(err, "failed to commit generated resources to git in dir %s", outDir)
	}
	return nil
}

// Git returns the gitter - lazily creating one if required
func (o *TemplateOptions) Git() gitclient.Interface {
	if o.Gitter == nil {
		o.Gitter = cli.NewCLIClient("", o.CommandRunner)
	}
	return o.Gitter
}
