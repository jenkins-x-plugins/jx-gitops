package add

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/apis/gitops/v1alpha1"
	helmfileadd "github.com/jenkins-x-plugins/jx-gitops/pkg/cmd/helmfile/add"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/helmhelpers"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/sourceconfigs"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/naming"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Adds a new Jenkins server to the git repository
`)

	cmdExample = templates.Examples(`
		# adds a new jenkins server to the git repository
		%s jenkins add --name myjenkins

	`)

	sampleValuesFile = `# custom Jenkins chart configuration
# see https://github.com/jenkinsci/helm-charts/blob/main/charts/jenkins/VALUES_SUMMARY.md

sampleValue: removeMeWhenYouAddRealConfiguration
`
)

// Options the options for the command
type Options struct {
	helmfileadd.Options
	Name string
}

// NewCmdJenkinsAdd creates a command object for the command
func NewCmdJenkinsAdd() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "add",
		Aliases: []string{"create", "new"},
		Short:   "Adds a new Jenkins server to the git repository",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Name, "name", "n", "", "the name of the jenkins server to add")
	cmd.Flags().StringVarP(&o.Chart, "chart", "c", "jenkinsci/jenkins", "the jenkins helm chart to use")
	cmd.Flags().StringVarP(&o.Repository, "repository", "r", "https://charts.jenkins.io", "the helm chart repository URL of the chart")
	cmd.Flags().StringVarP(&o.Version, "version", "v", "", "the version of the helm chart. If not specified the versionStream will be checked otherwise the latest version is used")
	return cmd, o
}

func (o *Options) Run() error {
	o.Name = strings.TrimSpace(o.Name)
	if o.Name == "" {
		return options.MissingOption("name")
	}
	o.Name = naming.ToValidName(o.Name)
	o.Namespace = o.Name
	o.Options.ReleaseName = "jenkins"
	o.Helmfile = filepath.Join(o.Dir, "helmfiles", o.Namespace, "helmfile.yaml")

	err := o.verifyValuesExists()
	if err != nil {
		return errors.Wrapf(err, "failed to verify values file exists")
	}

	err = o.Options.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to add jenkins helm chart for %s", o.Name)
	}
	log.Logger().Infof("added helmfile %s for jenkins server %s", info(o.Helmfile), info(o.Name))

	// lets add the jenkins-resources chart too
	o.Options.Chart = "jx3/jenkins-resources"
	o.Options.ReleaseName = "jenkins-resources"
	o.Options.Repository = helmhelpers.JX3HelmRepository
	o.Options.Values = nil
	err = o.Options.Run()
	if err != nil {
		return errors.Wrapf(err, "failed to add jenkins resources helm chart for %s", o.Name)
	}

	// lets make sure that there's a jenkins server of this name in the source config
	srcConfig, err := sourceconfigs.LoadSourceConfig(o.Dir, false)
	if err != nil {
		return errors.Wrapf(err, "failed to load source config")
	}
	for _, s := range srcConfig.Spec.JenkinsServers {
		if s.Server == o.Name {
			return nil
		}
	}
	srcConfig.Spec.JenkinsServers = append(srcConfig.Spec.JenkinsServers, v1alpha1.JenkinsServer{
		Server: o.Name,
	})
	err = sourceconfigs.SaveSourceConfig(srcConfig, o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to save source config")
	}
	_, err = o.Git().Command(o.Dir, "add", ".jx")
	if err != nil {
		return errors.Wrapf(err, "failed to add source config changes to git in dir %s", o.Dir)
	}
	return nil
}

func (o *Options) verifyValuesExists() error {
	if len(o.Values) == 0 {
		o.Values = []string{"values.yaml"}
	}
	if stringhelpers.StringArrayIndex(o.Values, "values.yaml") < 0 {
		return nil
	}

	// lets check if there's a values.yaml file and if not create one
	outDir := filepath.Join(o.Dir, "helmfiles", o.Namespace)

	err := os.MkdirAll(outDir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to make dir %s", outDir)
	}

	path := filepath.Join(outDir, "values.yaml")
	exists, err := files.FileExists(path)
	if err != nil {
		return errors.Wrapf(err, "failed to check if file exists %s", path)
	}
	if exists {
		return nil
	}

	err = ioutil.WriteFile(path, []byte(sampleValuesFile), files.DefaultFileWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save %s", path)
	}
	return nil
}
