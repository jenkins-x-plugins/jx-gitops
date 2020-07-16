package jx_apps

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/helm"
	"github.com/jenkins-x/jx-gitops/pkg/cmd/jx_apps/templater"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/pkg/versionstream/versionstreamrepo"

	"github.com/jenkins-x/jx-apps/pkg/jxapps"

	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/versionstream"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	jxAppsTemplateLong = templates.LongDesc(`
		Generate the kubernetes resources from a jx-apps.yml
`)

	jxAppsTemplateExample = templates.Examples(`
		# generates the resources from a jx-apps.yml
		%s step jx-apps template
	`)
)

var (
	phases = []string{"apps", "system"}
)

// JxAppsTemplateOptions the options for the command
type JxAppsTemplateOptions struct {
	helm.TemplateOptions
	Dir                 string
	VersionStreamDir    string
	VersionStreamURL    string
	VersionStreamRef    string
	TemplateValuesFiles []string
	prefixes            *versionstream.RepositoryPrefixes
	IOFileHandles       *files.IOFileHandles
}

// NewCmdJxAppsTemplate creates a command object for the command
func NewCmdJxAppsTemplate() (*cobra.Command, *JxAppsTemplateOptions) {
	o := &JxAppsTemplateOptions{}

	cmd := &cobra.Command{
		Use:     "template",
		Short:   "Generate the kubernetes resources from a jx-apps.yml",
		Long:    jxAppsTemplateLong,
		Example: fmt.Sprintf(jxAppsTemplateExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.OutDir, "output-dir", "o", "", "the output directory to generate the templates to. Defaults to charts/$name/resources")
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory that contains the jx-apps.yml")
	cmd.Flags().StringVarP(&o.VersionStreamDir, "version-stream-dir", "", "", "optional directory that contains a version stream")
	cmd.Flags().StringVarP(&o.GitCommitMessage, "commit-message", "", "chore: generated kubernetes resources from helm chart", "the git commit message used")
	cmd.Flags().StringVarP(&o.VersionStreamURL, "url", "n", "", "the git clone URL of the version stream")
	cmd.Flags().StringVarP(&o.VersionStreamRef, "ref", "c", "master", "the git ref (branch, tag, revision) to git clone")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", "jx", "the default namespace if none is specified in the jx-apps.yml or jx-requirements.yml")
	cmd.Flags().StringArrayVarP(&o.TemplateValuesFiles, "template-values", "", nil, "provide extra values.yaml files passed into evaluating any values.yaml.gotmpl files such as for generating dummy secret values")
	o.TemplateOptions.AddFlags(cmd)
	return cmd, o
}

// Run implements the command
func (o *JxAppsTemplateOptions) Run() error {
	appsCfg, appsCfgFile, err := jxapps.LoadAppConfig(o.Dir)
	if err != nil {
		return errors.Wrap(err, "failed to load jx-apps.yml")
	}

	outDir := o.OutDir
	if outDir == "" {
		outDir = "config-root"
	}

	err = os.MkdirAll(outDir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure output directory exists %s", outDir)
	}
	// todo should we get the version stream here? this isn't quite right as we need the jx apps
	versionsDir := o.VersionStreamDir
	requirements, _, err := config.LoadRequirementsConfig(o.Dir, false)
	if err != nil {
		return errors.Wrapf(err, "failed to load jx-requirements.yml")
	}
	if o.VersionStreamDir == "" {
		if o.VersionStreamURL == "" {
			o.VersionStreamURL = requirements.VersionStream.URL
		}
		if o.VersionStreamURL == "" {
			return errors.Errorf("Missing option:  --%s ", termcolor.ColorInfo("url"))
		}

		var err error
		o.Dir, err = ioutil.TempDir("", "jx-version-stream-")
		if err != nil {
			return errors.Wrap(err, "failed to create temp dir")
		}

		versionsDir, _, err = versionstreamrepo.CloneJXVersionsRepoToDir(o.Dir, o.VersionStreamURL, o.VersionStreamRef, nil, o.Git(), true, false, files.GetIOFileHandles(o.IOFileHandles))
		if err != nil {
			return errors.Wrapf(err, "failed to clone version stream to %s", o.Dir)
		}
	}
	if o.GitCommitMessage == "" {
		o.GitCommitMessage = "chore: generated kubernetes resources from helm charts"
	}

	resolver := &versionstream.VersionResolver{
		VersionsDir: versionsDir,
	}
	o.prefixes, err = resolver.GetRepositoryPrefixes()
	if err != nil {
		return errors.Wrapf(err, "failed to load repository prefixes at %s", versionsDir)
	}

	absVersionDir, err := filepath.Abs(versionsDir)
	if err != nil {
		return errors.Wrapf(err, "failed to find the absolute dir for %s", versionsDir)
	}

	appsCfgDir := filepath.Dir(appsCfgFile)

	jxReqValuesFile, err := ioutil.TempFile("", "jx-req-values-yaml-")
	if err != nil {
		return errors.Wrap(err, "failed to create tempo file for jx requirements values")
	}
	jxReqValuesFileName := jxReqValuesFile.Name()
	err = SaveRequirementsValuesFile(requirements, jxReqValuesFileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save tempo file for jx requirements values file %s", jxReqValuesFileName)
	}

	count := 0
	for _, app := range appsCfg.Apps {
		repository := app.Repository
		fullChartName := app.Name
		parts := strings.Split(app.Name, "/")
		if len(parts) != 2 {
			return errors.Wrapf(err, "failed to find prefix in the form prefix/name from app name %s", app.Name)
		}
		prefix := parts[0]
		chartName := parts[1]

		// lets resolve the chart prefix from a local repository from the file or from a
		// prefix in the versions stream
		if repository == "" && prefix != "" {
			for _, r := range appsCfg.Repositories {
				if r.Name == prefix {
					repository = r.URL
				}
			}
		}
		if repository == "" && prefix != "" {
			repository, err = o.matchPrefix(prefix)
			if err != nil {
				return errors.Wrapf(err, "failed to match prefix %s with repositories from versionstream %s", prefix, o.VersionStreamURL)
			}
		}
		if repository == "" && prefix != "" {
			return errors.Wrapf(err, "failed to find repository URL, not defined in jx-apps.yml or versionstream %s", o.VersionStreamURL)
		}

		version, err := resolver.StableVersionNumber(versionstream.KindChart, fullChartName)
		if err != nil {
			return errors.Wrapf(err, "failed to find version number for chart %s", fullChartName)
		}

		defaultsDir := filepath.Join(versionsDir, string(versionstream.KindApp), fullChartName)
		defaults, _, err := jxapps.LoadAppDefaultsConfig(defaultsDir)
		if err != nil {
			return errors.Wrapf(err, "failed to load defaults from dir %s", defaultsDir)
		}

		if version == "" {
			log.Logger().Warnf("could not find version for chart %s so using latest found in helm repository %s", fullChartName, repository)
		}

		ho := o.TemplateOptions
		ho.Gitter = o.Git()
		ho.GitCommitMessage = o.GitCommitMessage
		ho.DoGitCommit = false
		ho.Version = version
		ho.Chart = chartName

		ho.Namespace = app.Namespace
		if ho.Namespace == "" && appsCfg.DefaultNamespace != "" {
			ho.Namespace = appsCfg.DefaultNamespace
		}

		if ho.Namespace == "" && defaults.Namespace != "" {
			ho.Namespace = defaults.Namespace
		}

		if ho.Namespace == "" {
			ho.Namespace = requirements.Cluster.Namespace
			if ho.Namespace == "" {
				ho.Namespace = o.Namespace
			}
		}

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

		ho.Repository = repository
		ho.ValuesFiles = append(ho.ValuesFiles, jxReqValuesFileName)

		valuesDir := filepath.Join(absVersionDir, "charts", prefix, chartName)
		err = os.MkdirAll(valuesDir, files.DefaultDirWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to create values dir for chart %s", fullChartName)
		}

		verisonStreamAppsDir := filepath.Join(absVersionDir, "apps")
		foundAppsFile := false
		appValuesFile := filepath.Join(verisonStreamAppsDir, prefix, chartName, "values.yaml")
		exists, err := files.FileExists(appValuesFile)
		if err != nil {
			return errors.Wrapf(err, "failed to check if app values file exists %s", appValuesFile)
		}
		if exists {
			foundAppsFile = true
			ho.ValuesFiles = append(ho.ValuesFiles, appValuesFile)
		}

		appValuesFile = filepath.Join(verisonStreamAppsDir, prefix, chartName, "values.yaml.gotmpl")
		exists, err = files.FileExists(appValuesFile)
		if err != nil {
			return errors.Wrapf(err, "failed to check if app values file exists %s", appValuesFile)
		}
		if exists {
			tmpFilePrefix := strings.ReplaceAll(chartName, "/", "-")
			generatedValuesFile, err := o.templateValuesFile(requirements, appValuesFile, tmpFilePrefix, o.TemplateValuesFiles)
			if err != nil {
				return errors.Wrapf(err, "failed to generate templated values file %s", appValuesFile)
			}
			foundAppsFile = true
			ho.ValuesFiles = append(ho.ValuesFiles, generatedValuesFile)
		}

		if !foundAppsFile {
			templateValuesFile := filepath.Join(valuesDir, "template-values.yaml")
			exists, err := files.FileExists(templateValuesFile)
			if err != nil {
				return errors.Wrapf(err, "failed to check if template values file exists %s", templateValuesFile)
			}

			if exists {
				ho.ValuesFiles = append(ho.ValuesFiles, templateValuesFile)
			}
		}

		for _, phase := range phases {

			// find any extra values files
			valuesFile := filepath.Join(appsCfgDir, phase, ho.ReleaseName, "values.yaml")
			exists, err = files.FileExists(valuesFile)
			if err != nil {
				return errors.Wrapf(err, "failed to find values file %s", valuesFile)
			}
			if exists {
				absValuesFile, err := filepath.Abs(valuesFile)
				if err != nil {
					return errors.Wrapf(err, "failed to get absolute path of %s", valuesFile)
				}
				ho.ValuesFiles = append(ho.ValuesFiles, absValuesFile)
			}

			// find any extra gotmpl values files in each phase
			appValuesFile = filepath.Join(appsCfgDir, phase, ho.ReleaseName, "values.yaml.gotmpl")
			exists, err = files.FileExists(appValuesFile)
			if err != nil {
				return errors.Wrapf(err, "failed to check if app values file exists %s", appValuesFile)
			}
			if exists {
				tmpFilePrefix := strings.ReplaceAll(ho.ReleaseName, "/", "-")
				generatedValuesFile, err := o.templateValuesFile(requirements, appValuesFile, tmpFilePrefix, o.TemplateValuesFiles)
				if err != nil {
					return errors.Wrapf(err, "failed to generate templated values file %s", appValuesFile)
				}
				foundAppsFile = true
				ho.ValuesFiles = append(ho.ValuesFiles, generatedValuesFile)
			}
		}

		log.Logger().Infof("generating chart %s version %s to dir %s", fullChartName, version, ho.OutDir)

		err = ho.Run()
		if err != nil {
			return errors.Wrapf(err, "failed to helm template chart %s version %s to dir %s", fullChartName, version, ho.OutDir)
		}
		count++
	}

	log.Logger().Infof("processed %d charts", count)

	if !o.TemplateOptions.DoGitCommit {
		return nil
	}
	if count > 0 {
		log.Logger().Infof("committing changes: %s", o.GitCommitMessage)
		err = o.TemplateOptions.GitCommit(outDir, o.GitCommitMessage)
		if err != nil {
			log.Logger().Warnf("failed to commit in dir %s due to: %s", outDir, err.Error())
		}
	}
	return nil

}

func (o *JxAppsTemplateOptions) matchPrefix(prefix string) (string, error) {
	if o.prefixes == nil {
		return "", errors.Errorf("no repository prefixes found in version stream")
	}
	// default to first URL
	repoURL := o.prefixes.URLsForPrefix(prefix)

	if repoURL == nil || len(repoURL) == 0 {
		return "", errors.Errorf("no matching repository for for prefix %s", prefix)
	}
	return repoURL[0], nil
}

func (o *JxAppsTemplateOptions) templateValuesFile(requirements *config.RequirementsConfig, valuesTemplateFile string, chartName string, valuesFiles []string) (string, error) {
	absValuesFiles := []string{}
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
