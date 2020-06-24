package jx_apps

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/helm"

	"github.com/jenkins-x/jx-promote/pkg/versionstream/versionstreamrepo"

	"github.com/jenkins-x/jx-promote/pkg/jxapps"

	"github.com/jenkins-x/jx-gitops/pkg/common"
	"github.com/jenkins-x/jx-promote/pkg/versionstream"
	"github.com/jenkins-x/jx/v2/pkg/cmd/helper"
	"github.com/jenkins-x/jx/v2/pkg/cmd/templates"
	"github.com/jenkins-x/jx/v2/pkg/config"
	"github.com/jenkins-x/jx/v2/pkg/gits"
	"github.com/jenkins-x/jx/v2/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/util"
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

const (
	pathSeparator = string(os.PathSeparator)
)

// JxAppsTemplateOptions the options for the command
type JxAppsTemplateOptions struct {
	helm.TemplateOptions
	OutDir           string
	Dir              string
	VersionStreamDir string
	DefaultDomain    string
	GitCommitMessage string
	VersionStreamURL string
	VersionStreamRef string
	NoGitCommit      bool
	NoSplit          bool
	NoExtSecrets     bool
	IncludeCRDs      bool
	Gitter           gits.Gitter
	prefixes         *versionstream.RepositoryPrefixes
	IOFileHandles    *util.IOFileHandles
}

// NewCmdJxAppsTemplate creates a command object for the command
func NewCmdJxAppsTemplate() (*cobra.Command, *JxAppsTemplateOptions) {
	o := &JxAppsTemplateOptions{}

	cmd := &cobra.Command{
		Use:     "template",
		Short:   "Generate the kubernetes resources from a jx-apps.yml",
		Long:    jxAppsTemplateLong,
		Example: fmt.Sprintf(jxAppsTemplateExample, common.BinaryName),
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
	o.AddFlags(cmd)
	return cmd, o
}

func (o *JxAppsTemplateOptions) AddFlags(cmd *cobra.Command) {
	cmd.Flags().StringVarP(&o.DefaultDomain, "domain", "", "cluster.local", "the default domain name in the generated ingress")
	cmd.Flags().BoolVarP(&o.NoGitCommit, "no-git-commit", "", false, "if set then the command will not git add/commit the generated resources")
	cmd.Flags().BoolVarP(&o.NoSplit, "no-split", "", false, "if set then disable splitting of multiple resources into separate files")
	cmd.Flags().BoolVarP(&o.NoExtSecrets, "no-external-secrets", "", false, "if set then disable converting Secret resources to ExternalSecrets")
	cmd.Flags().BoolVarP(&o.IncludeCRDs, "include-crds", "", true, "if CRDs should be included in the output")
}

// Run implements the command
func (o *JxAppsTemplateOptions) Run() error {

	appsCfg, _, err := jxapps.LoadAppConfig(o.Dir)
	if err != nil {
		return errors.Wrap(err, "failed to load jx-apps.yml")
	}

	outDir := o.OutDir
	if outDir == "" {
		outDir = "config-root"
	}

	err = os.MkdirAll(outDir, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to ensure output directory exists %s", outDir)
	}
	// todo should we get the version stream here? this isn't quite right as we need the jx apps
	versionsDir := o.VersionStreamDir
	if o.VersionStreamDir == "" {
		if o.VersionStreamURL == "" {
			requirements, _, err := config.LoadRequirementsConfig(o.Dir, false)
			if err != nil {
				return errors.Wrapf(err, "failed to load jx-requirements.yml")
			}
			o.VersionStreamURL = requirements.VersionStream.URL
		}
		if o.VersionStreamURL == "" {
			return errors.Errorf("Missing option:  --%s ", util.ColorInfo("url"))
		}

		var err error
		o.Dir, err = ioutil.TempDir("", "jx-version-stream-")
		if err != nil {
			return errors.Wrap(err, "failed to create temp dir")
		}

		versionsDir, _, err = versionstreamrepo.CloneJXVersionsRepoToDir(o.Dir, o.VersionStreamURL, o.VersionStreamRef, nil, o.Git(), true, false, common.GetIOFileHandles(o.IOFileHandles))
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

		if repository == "" && prefix != "" {
			repository, err = o.matchPrefix(prefix)
			if err != nil {
				return errors.Wrapf(err, "failed to match prefix %s with repositories from versionstream %s", prefix, o.VersionStreamURL)
			}
		} else {
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
		ho.Version = version
		ho.Chart = chartName

		ho.Namespace = app.Namespace
		if ho.Namespace == "" && appsCfg.DefaultNamespace != "" {
			ho.Namespace = appsCfg.DefaultNamespace
		}

		if ho.Namespace == "" && defaults.Namespace != "" {
			ho.Namespace = defaults.Namespace
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

		valuesDir := filepath.Join(absVersionDir, "charts", prefix, chartName)
		err = os.MkdirAll(valuesDir, util.DefaultWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to create values dir for chart %s", fullChartName)
		}

		templateValuesFile := filepath.Join(valuesDir, "template-values.yaml")
		exists, err := util.FileExists(templateValuesFile)
		if err != nil {
			return errors.Wrapf(err, "failed to check if template values file exists %s", templateValuesFile)
		}

		if exists {
			ho.ValuesFiles = append(ho.ValuesFiles, templateValuesFile)
		}

		appSubfolder := "apps"
		if app.Phase != "" {
			appSubfolder = string(app.Phase)
		}

		absDir, err := filepath.Abs(o.Dir)
		if err != nil {
			return errors.Wrapf(err, "failed to find the absolute dir for %s", o.Dir)
		}

		appValuesFile := filepath.Join(absDir, appSubfolder, ho.ReleaseName, "values.yaml")
		exists, err = util.FileExists(appValuesFile)
		if err != nil {
			return errors.Wrapf(err, "failed to check if app values file exists %s", appValuesFile)
		}

		if exists {
			ho.ValuesFiles = append(ho.ValuesFiles, appValuesFile)
		}

		log.Logger().Infof("generating chart %s version %s to dir %s", fullChartName, version, ho.OutDir)

		err = ho.Run()
		if err != nil {
			return errors.Wrapf(err, "failed to helm template chart %s version %s to dir %s", fullChartName, version, ho.OutDir)
		}
		count++
	}

	log.Logger().Infof("processed %d charts", count)

	if count > 0 {
		err = o.TemplateOptions.GitCommit(outDir, o.GitCommitMessage)
		if err != nil {
			log.Logger().Warnf("failed to commit in dir %s due to: %s", outDir, err.Error())
		}
	}
	return nil

}

func (o *JxAppsTemplateOptions) GitCommit(outDir string, commitMessage string) error {
	gitter := o.Git()
	err := gitter.Add(outDir, "*")
	if err != nil {
		return errors.Wrapf(err, "failed to add generated resources to git in dir %s", outDir)
	}
	err = gitter.CommitIfChanges(outDir, commitMessage)
	if err != nil {
		return errors.Wrapf(err, "failed to commit generated resources to git in dir %s", outDir)
	}
	return nil
}

// Git returns the gitter - lazily creating one if required
func (o *JxAppsTemplateOptions) Git() gits.Gitter {
	if o.Gitter == nil {
		o.Gitter = gits.NewGitCLI()
	}
	return o.Gitter
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
