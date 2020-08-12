package template

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/apps/resolve"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/helm"
	"github.com/jenkins-x/jx-gitops/pkg/jxtmpl/templater"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	cmdLong = templates.LongDesc(`
		Generate the kubernetes resources from a jx-apps.yml
`)

	cmdExample = templates.Examples(`
		# generates the resources from a jx-apps.yml
		%s step apps template
	`)
)

// Options the options for the command
type Options struct {
	resolve.Options
	helm.TemplateOptions
	TemplateValuesFiles []string
	NoResolve           bool
}

// NewCmdJxAppsTemplate creates a command object for the command
func NewCmdJxAppsTemplate() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "template",
		Short:   "Generate the kubernetes resources from a jx-apps.yml",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.OutDir, "output-dir", "o", "", "the output directory to generate the templates to. Defaults to charts/$name/resources")
	cmd.Flags().BoolVarP(&o.NoResolve, "no-resolve", "", false, "disables running the resolve command to resolve any versions of values from the version stream in jx-apps.yml")
	cmd.Flags().StringArrayVarP(&o.TemplateValuesFiles, "template-values", "", nil, "provide extra values.yaml files passed into evaluating any values.yaml.gotmpl files before passing them into Helm - such as for generating dummy secret values")

	o.Options.AddFlags(cmd, "apps-")
	o.TemplateOptions.AddFlags(cmd)
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	outDir := o.OutDir
	if outDir == "" {
		outDir = "config-root"
	}
	if o.TemplateOptions.GitCommitMessage == "" {
		o.TemplateOptions.GitCommitMessage = "chore: generated kubernetes resources from helm charts"
	}

	ro := &o.Options
	err := ro.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate")
	}

	err = os.MkdirAll(outDir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure output directory exists %s", outDir)
	}

	if !o.NoResolve {
		err = ro.Run()
		if err != nil {
			return errors.Wrapf(err, "failed to resolve versions and helm values files in jx-apps.yml file")
		}
	}

	appsCfg := o.Results.AppsCfg
	if appsCfg == nil {
		return errors.Errorf("failed to load the jx-apps.yml")
	}
	requirements := ro.Results.Requirements
	if requirements == nil {
		return errors.Errorf("failed to load the jx-requirements.yml")
	}
	count := 0
	for _, app := range appsCfg.Apps {
		repository := app.Repository
		fullChartName := app.Name
		parts := strings.Split(app.Name, "/")
		if len(parts) != 2 {
			return errors.Wrapf(err, "failed to find prefix in the form prefix/name from app name %s", app.Name)
		}
		chartName := parts[1]
		prefix := parts[0]
		if repository == "" {
			// lets find the repository URL from the repos
			for _, r := range appsCfg.Repositories {
				if r.Name == prefix {
					repository = r.URL
					break
				}
			}
			if repository == "" {
				return errors.Errorf("could not find chart repository URL for prefix %s chart %s", prefix, chartName)
			}
		}
		ho := o.TemplateOptions
		ho.Repository = repository
		ho.Gitter = o.TemplateOptions.Git()
		ho.GitCommitMessage = o.TemplateOptions.GitCommitMessage
		ho.DoGitCommit = false
		ho.Version = app.Version
		ho.Chart = chartName
		ho.Namespace = app.Namespace
		if ho.Namespace != "" {
			ho.OutDir = filepath.Join(outDir, ho.Namespace, chartName)
		} else {
			ho.OutDir = filepath.Join(outDir, chartName)
		}

		if app.Alias != "" {
			ho.ReleaseName = app.Alias
		} else {
			ho.ReleaseName = chartName
		}

		ho.ValuesFiles, err = o.addAbsValuesFile(ho.ValuesFiles, appsCfg.Values...)
		if err != nil {
			return errors.Wrapf(err, "failed to add absolute values file paths")
		}

		for _, valuesFile := range app.Values {
			if strings.HasSuffix(valuesFile, ".gotmpl") {
				absValuesFile, err := filepath.Abs(filepath.Join(o.Dir, valuesFile))
				if err != nil {
					return errors.Wrapf(err, "failed to get absolute path of %s", valuesFile)
				}

				tmpFilePrefix := strings.ReplaceAll(ho.ReleaseName, "/", "-")
				generatedValuesFile, err := o.templateValuesFile(requirements, absValuesFile, tmpFilePrefix, o.TemplateValuesFiles)
				if err != nil {
					return errors.Wrapf(err, "failed to generate values file from go template: %s", absValuesFile)
				}
				ho.ValuesFiles = append(ho.ValuesFiles, generatedValuesFile)
			} else {
				ho.ValuesFiles, err = o.addAbsValuesFile(ho.ValuesFiles, valuesFile)
				if err != nil {
					return errors.Wrapf(err, "failed to add absolute values file paths")
				}
			}
		}
		err = ho.Run()
		if err != nil {
			return errors.Wrapf(err, "failed to helm template chart %s version %s to dir %s", fullChartName, app.Version, ho.OutDir)
		}
		count++
	}

	log.Logger().Infof("processed %d charts", count)

	if !o.TemplateOptions.DoGitCommit {
		return nil
	}
	if count > 0 {
		log.Logger().Infof("committing changes: %s", o.TemplateOptions.GitCommitMessage)
		err = o.TemplateOptions.GitCommit(outDir, o.TemplateOptions.GitCommitMessage)
		if err != nil {
			log.Logger().Warnf("failed to commit in dir %s due to: %s", outDir, err.Error())
		}
	}
	return nil

}

func (o *Options) addAbsValuesFile(valuesFiles []string, files ...string) ([]string, error) {
	for _, f := range files {
		path, err := filepath.Abs(filepath.Join(o.Dir, f))
		absValuesFile := path
		if err != nil {
			return nil, errors.Wrapf(err, "failed to get absolute path of %s", path)
		}

		if stringhelpers.StringArrayIndex(valuesFiles, absValuesFile) < 0 {
			valuesFiles = append(valuesFiles, absValuesFile)
		}
	}
	return valuesFiles, nil
}

func (o *Options) templateValuesFile(requirements *config.RequirementsConfig, valuesTemplateFile string, chartName string, valuesFiles []string) (string, error) {
	var absValuesFiles []string
	for _, f := range valuesFiles {
		af, err := filepath.Abs(f)
		if err != nil {
			return "", errors.Wrapf(err, "failed to find the absolute file for %s", f)
		}
		absValuesFiles = append(absValuesFiles, af)
	}

	t := templater.NewTemplater(requirements, absValuesFiles)
	log.Logger().Infof("templating the values file %s", termcolor.ColorInfo(absValuesFiles))

	tmpFile, err := ioutil.TempFile("", chartName+"-")
	if err != nil {
		return "", errors.Wrapf(err, "failed to create temp file for values template %s", valuesTemplateFile)
	}
	tmpFileName := tmpFile.Name()

	err = t.Generate(valuesTemplateFile, tmpFileName)
	if err != nil {
		return tmpFileName, errors.Wrapf(err, "failed to template file %s", valuesTemplateFile)
	}
	return tmpFileName, nil
}
