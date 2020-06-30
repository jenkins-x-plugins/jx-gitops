package helm

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-apps/pkg/jxapps"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/pkg/files"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/pkg/versionstream"
	"github.com/jenkins-x/jx-helpers/pkg/versionstream/versionstreamrepo"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	helmStreamLong = templates.LongDesc(`
		Generate the kubernetes resources for all helm charts in a version stream
`)

	helmStreamExample = templates.Examples(`
		%s step helm stream
	`)

	defaultValuesYaml = `jxRequirements:
  ingress:
    domain: %s
    namespaceSubDomain: "."
    tls:
      enabled: true 
`

	pathSeparator = string(os.PathSeparator)
)

// HelmStreamOptions the options for the command
type StreamOptions struct {
	TemplateOptions

	Dir              string
	VersionStreamURL string
	VersionStreamRef string
	IOFileHandles    *files.IOFileHandles
}

// NewCmdHelmStream creates a command object for the command
func NewCmdHelmStream() (*cobra.Command, *StreamOptions) {
	o := &StreamOptions{}

	cmd := &cobra.Command{
		Use:     "stream",
		Short:   "Generate the kubernetes resources for all helm charts in a version stream",
		Long:    helmStreamLong,
		Example: fmt.Sprintf(helmStreamExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.OutDir, "output-dir", "o", ".", "the output directory to generate the templates to")
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", "", "the directory to look for the version stream git clone")
	cmd.Flags().StringVarP(&o.VersionStreamURL, "url", "n", "", "the git clone URL of the version stream")
	cmd.Flags().StringVarP(&o.VersionStreamRef, "ref", "c", "master", "the git ref (branch, tag, revision) to git clone")
	cmd.Flags().StringVarP(&o.GitCommitMessage, "commit-message", "", "", "the git commit message used")

	ho := &o.TemplateOptions
	ho.AddFlags(cmd)
	return cmd, o
}

// Run implements the command
func (o *StreamOptions) Run() error {
	versionsDir := o.Dir
	if o.Dir == "" {
		if o.VersionStreamURL == "" {
			return errors.Errorf("Missing option: --%s or --%s ", util.ColorInfo("dir"), util.ColorInfo("url"))
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

	chartsDir := filepath.Join(versionsDir, "charts")
	exists, err := util.DirExists(chartsDir)
	if err != nil {
		return errors.Wrapf(err, "failed to check of charts dir %s exists", chartsDir)
	}
	if !exists {
		return errors.Errorf("dir %s does not exist in version stream", chartsDir)
	}

	resolver := &versionstream.VersionResolver{
		VersionsDir: versionsDir,
	}
	prefixes, err := resolver.GetRepositoryPrefixes()
	if err != nil {
		return errors.Wrapf(err, "failed to load repository prefixes at %s", versionsDir)
	}
	if prefixes == nil {
		return errors.Errorf("no repository prefixes found at %s", versionsDir)
	}
	absVersionDir, err := filepath.Abs(versionsDir)
	if err != nil {
		return errors.Wrapf(err, "failed to find the absolute dir for %s", versionsDir)
	}

	outDir := o.OutDir
	count := 0
	err = filepath.Walk(chartsDir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yml") {
			return nil
		}

		rel, err := filepath.Rel(chartsDir, path)
		if err != nil {
			return errors.Wrapf(err, "failed to find relative path of %s from %s", path, chartsDir)
		}

		chartName := strings.TrimSuffix(rel, ".yml")
		if chartName == "repositories" {
			// ignore the top level repositories.yml
			return nil
		}

		version, err := resolver.StableVersionNumber(versionstream.KindChart, chartName)
		if err != nil {
			return errors.Wrapf(err, "failed to find version number for chart %s", chartName)
		}

		defaultsDir := filepath.Join(versionsDir, string(versionstream.KindApp), chartName)
		defaults, _, err := jxapps.LoadAppDefaultsConfig(defaultsDir)
		if err != nil {
			return errors.Wrapf(err, "failed to load defaults from dir %s", defaultsDir)
		}

		if version == "" {
			return fmt.Errorf("could not find version for chart %s", chartName)
		}

		chartOutput := filepath.Join(outDir, chartName)
		ho := o.TemplateOptions
		ho.Gitter = o.Git()
		ho.OutDir = chartOutput
		ho.Version = version

		// lets avoid using the charts dir to run 'helm template' as 'flagger' is a repo name and a chart name which confuses 'helm template'
		_, ho.ReleaseName = filepath.Split(chartName)

		// lets use the chart name within the chart repository
		ho.Chart = ho.ReleaseName

		if defaults.Namespace != "" {
			ho.Namespace = defaults.Namespace
		}

		// lets find the repository prefix
		paths := strings.Split(chartName, pathSeparator)
		repoPrefix := paths[0]
		repoURLs := prefixes.URLsForPrefix(repoPrefix)
		if len(repoURLs) == 0 {
			return errors.Errorf("could not find repository prefix %s in the repositories.yml file in the version stream", repoPrefix)
		}
		ho.Repository = repoURLs[0]

		valuesDir := filepath.Join(absVersionDir, "charts", chartName)
		err = os.MkdirAll(valuesDir, util.DefaultWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to create values dir for chart %s", chartName)
		}
		templateValuesFile := filepath.Join(valuesDir, "template-values.yaml")
		err = o.lazyCreateValuesFile(templateValuesFile)
		if err != nil {
			return errors.Wrapf(err, "failed to lazily create the template values file")
		}

		ho.ValuesFiles = append(ho.ValuesFiles, templateValuesFile)

		log.Logger().Infof("generating chart %s version %s to dir %s", chartName, version, chartOutput)

		err = ho.Run()
		if err != nil {
			return errors.Wrapf(err, "failed to helm template chart %s version %s to dir %s", chartName, version, chartOutput)
		}

		count++
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to process charts dir %s", chartsDir)
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

func (o *StreamOptions) lazyCreateValuesFile(valuesFile string) error {
	exists, err := util.FileExists(valuesFile)
	if err != nil {
		return errors.Wrapf(err, "failed to check if values file exists %s", valuesFile)
	}
	if !exists {
		text := fmt.Sprintf(defaultValuesYaml, o.DefaultDomain)
		dir := filepath.Dir(valuesFile)
		if dir != "" && dir != "." {
			err = os.MkdirAll(dir, util.DefaultWritePermissions)
			if err != nil {
				return errors.Wrapf(err, "failed to ensure that values file directory %s can be created", dir)
			}
		}
		err = ioutil.WriteFile(valuesFile, []byte(text), util.DefaultFileWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to save default values file %s", valuesFile)
		}
	}
	return err
}

// Git returns the gitter - lazily creating one if required
func (o *StreamOptions) Git() gitclient.Interface {
	if o.Gitter == nil {
		o.Gitter = cli.NewCLIClient("", o.CommandRunner)
	}
	return o.Gitter
}
