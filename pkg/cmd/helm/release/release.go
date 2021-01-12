package release

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	jxc "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-gitops/pkg/plugins"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-gitops/pkg/variablefinders"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Generate the kubernetes resources from a helm chart
`)

	cmdExample = templates.Examples(`
		# generates the resources from a helm chart
		%s step helm template
	`)
)

// Options the options for the command
type Options struct {
	UseHelmPlugin        bool
	NoRelease            bool
	ChartOCI             bool
	HelmBinary           string
	ChartsDir            string
	RepositoryName       string
	RepositoryURL        string
	RepositoryUsername   string
	RepositoryPassword   string
	Version              string
	VersionFile          string
	Namespace            string
	ContainerRegistryOrg string
	KubeClient           kubernetes.Interface
	JXClient             jxc.Interface
	GitClient            gitclient.Interface
	CommandRunner        cmdrunner.CommandRunner
	Requirements         *jxcore.RequirementsConfig
}

// NewCmdHelmRelease creates a command object for the command
func NewCmdHelmRelease() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "release",
		Short:   "Performs a release of all the charts in the charts folder",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.ChartsDir, "charts-dir", "c", "charts", "the directory to look for helm charts to release")
	cmd.Flags().StringVarP(&o.RepositoryName, "repo-name", "n", "release-repo", "the name of the helm chart to release to. If not specified uses JX_CHART_REPOSITORY environment variable")
	cmd.Flags().StringVarP(&o.RepositoryURL, "repo-url", "u", "", "the URL to release to")
	cmd.Flags().StringVarP(&o.RepositoryUsername, "repo-username", "", "", "the username to access the chart repository. If not specified defaults to the environment variable $JX_REPOSITORY_USERNAME")
	cmd.Flags().StringVarP(&o.RepositoryPassword, "repo-password", "", "", "the password to access the chart repository. If not specified defaults to the environment variable $JX_REPOSITORY_PASSWORD")
	cmd.Flags().StringVarP(&o.Version, "version", "", "", "specify the version to release")
	cmd.Flags().StringVarP(&o.VersionFile, "version-file", "", "VERSION", "the file to load the version from if not specified directly or via a $VERSION environment variable")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", "", "the namespace to look for the dev Environment. Defaults to the current namespace")
	cmd.Flags().BoolVarP(&o.ChartOCI, "oci", "", false, "treat the repository as an OCI container registry. If not specified its defaulted from the cluster.chartOCI flag on the 'jx-requirements.yml' file")
	cmd.Flags().BoolVarP(&o.NoRelease, "no-release", "", false, "disables publishing the release. Useful for a Pull Request pipeline")
	cmd.Flags().BoolVarP(&o.UseHelmPlugin, "use-helm-plugin", "", false, "uses the jx binary plugin for helm rather than whatever helm is on the $PATH")
	return cmd, o
}

// Run implements the command
func (o *Options) Validate() error {
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.QuietCommandRunner
	}
	var err error
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
	if o.RepositoryUsername == "" {
		o.RepositoryUsername = os.Getenv("JX_REPOSITORY_USERNAME")
		if o.RepositoryUsername == "" {
			o.RepositoryUsername = os.Getenv("GITHUB_REPOSITORY_OWNER")
		}
	}
	if o.RepositoryPassword == "" {
		o.RepositoryPassword = os.Getenv("JX_REPOSITORY_PASSWORD")
		if o.RepositoryPassword == "" {
			o.RepositoryPassword = os.Getenv("GITHUB_TOKEN")
		}
	}

	o.JXClient, o.Namespace, err = jxclient.LazyCreateJXClientAndNamespace(o.JXClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to create jx client")
	}
	o.KubeClient, err = kube.LazyCreateKubeClient(o.KubeClient)
	if err != nil {
		return errors.Wrapf(err, "failed to create kube client")
	}

	// lets find the version
	if o.Version == "" {
		exists, err := files.FileExists(o.VersionFile)
		if err != nil {
			return errors.Wrapf(err, "failed to check for file %s", o.VersionFile)
		}
		if exists {
			data, err := ioutil.ReadFile(o.VersionFile)
			if err != nil {
				return errors.Wrapf(err, "failed to read version file %s", o.VersionFile)
			}
			o.Version = strings.TrimSpace(string(data))
		} else {
			log.Logger().Infof("version file %s does not exist", info(o.VersionFile))
		}
		if o.Version == "" {
			o.Version = os.Getenv("VERSION")
		}
		if o.Version == "" {
			return errors.Errorf("could not detect version from $VERSION or version file %s. Try supply the command option: --version", o.VersionFile)
		}
	}

	if o.GitClient == nil {
		o.GitClient = cli.NewCLIClient("", o.CommandRunner)
	}
	requirements, err := variablefinders.FindRequirements(o.JXClient, o.Namespace, o.GitClient)
	if err != nil {
		return errors.Wrapf(err, "failed to load requirements")
	}
	if requirements != nil {
		o.Requirements = requirements
		if requirements.Cluster.ChartOCI {
			o.ChartOCI = true
		}
		if o.ContainerRegistryOrg == "" {
			o.ContainerRegistryOrg = requirements.Cluster.DockerRegistryOrg
		}
	}
	return nil
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate")
	}
	dir := o.ChartsDir
	exists, err := files.DirExists(dir)
	if err != nil {
		return errors.Wrapf(err, "failed to check if charts dir exists %s", dir)
	}
	if !exists {
		log.Logger().Infof("no charts dir: %s", dir)
		return nil
	}
	fileSlice, err := ioutil.ReadDir(dir)
	if err != nil {
		return errors.Wrapf(err, "failed to read dir %s", dir)
	}
	count := 0
	for _, f := range fileSlice {
		if !f.IsDir() {
			continue
		}
		repoURL := o.RepositoryURL
		name := f.Name()
		chartDir := filepath.Join(dir, name)
		chartFile := filepath.Join(chartDir, "Chart.yaml")
		exists, err := files.FileExists(chartFile)
		if err != nil {
			return errors.Wrapf(err, "failed to check file exists %s", chartFile)
		}
		if !exists {
			continue
		}

		log.Logger().Infof("releasing chart %s", info(name))

		// find the repository URL
		if repoURL == "" {
			repoURL, err = variablefinders.FindRepositoryURL(o.JXClient, o.Namespace, o.Requirements, o.ContainerRegistryOrg, name)
			if err != nil {
				return errors.Wrapf(err, "failed to find chart repository URL")
			}
		}

		if o.ChartOCI {
			err = o.OCIRegistry(repoURL, chartDir, name)
		} else {
			err = o.BasicRegistry(repoURL, chartDir, name)
		}

		if err != nil {
			return errors.Wrapf(err, "failed to create release in dir %s", chartDir)
		}

		count++
	}

	log.Logger().Infof("released %d charts from the charts dir: %s", count, dir)
	return nil
}

func (o *Options) OCIRegistry(repoURL, chartDir, name string) error {
	qualifiedChartName := fmt.Sprintf("%s/%s:%s", repoURL, name, o.Version)

	c := &cmdrunner.Command{
		Dir:  chartDir,
		Name: o.HelmBinary,
		Env: map[string]string{
			"HELM_EXPERIMENTAL_OCI": "1",
		},
		Args: []string{"registry", "login", repoURL, "--username", o.RepositoryUsername, "--password", o.RepositoryPassword},
	}
	_, err := o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to login to registry %s for user %s", repoURL, o.RepositoryUsername)
	}

	c = &cmdrunner.Command{
		Dir:  chartDir,
		Name: o.HelmBinary,
		Env: map[string]string{
			"HELM_EXPERIMENTAL_OCI": "1",
		},
		Args: []string{"chart", "save", ".", qualifiedChartName},
	}
	_, err = o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to save chart %s in %s", qualifiedChartName, chartDir)
	}

	if o.NoRelease {
		log.Logger().Infof("disabling the chart publish")
		return nil
	}

	c = &cmdrunner.Command{
		Dir:  chartDir,
		Name: o.HelmBinary,
		Env: map[string]string{
			"HELM_EXPERIMENTAL_OCI": "1",
		},
		Args: []string{"chart", "push", qualifiedChartName},
	}
	_, err = o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to push chart %s", qualifiedChartName)
	}
	return nil
}

func (o *Options) BasicRegistry(repoURL, chartDir, name string) error {
	c := &cmdrunner.Command{
		Dir:  chartDir,
		Name: o.HelmBinary,
		Args: []string{"repo", "add", o.RepositoryName, repoURL},
	}
	_, err := o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to add remote repo")
	}

	c = &cmdrunner.Command{
		Dir:  chartDir,
		Name: o.HelmBinary,
		Args: []string{"dependency", "build", "."},
	}
	_, err = o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to build dependencies")
	}

	c = &cmdrunner.Command{
		Dir:  chartDir,
		Name: o.HelmBinary,
		Args: []string{"lint"},
	}
	_, err = o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to lint")
	}

	c = &cmdrunner.Command{
		Dir:  chartDir,
		Name: o.HelmBinary,
		Args: []string{"package", "."},
	}
	_, err = o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to package")
	}

	if o.NoRelease {
		log.Logger().Infof("disabling the chart publish")
		return nil
	}

	c, err = o.createPublishCommand(repoURL, name, chartDir)
	if err != nil {
		return errors.Wrapf(err, "failed to create release command in dir %s", chartDir)
	}

	_, err = o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to publish")
	}
	return nil
}

func (o *Options) createPublishCommand(repoURL, name, chartDir string) (*cmdrunner.Command, error) {
	tarFile := name + "-" + o.Version + ".tgz"

	if strings.HasPrefix(repoURL, "gs:") {
		// use gcs to push the chart
		return &cmdrunner.Command{
			Dir:  chartDir,
			Name: o.HelmBinary,
			Args: []string{"gcs", "push", tarFile, o.RepositoryName},
		}, nil
	}

	userSecret, err := o.findChartRepositoryUserPassword()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to find chart repository user:password")
	}

	url := stringhelpers.UrlJoin(repoURL, "/api/charts")

	return &cmdrunner.Command{
		Dir:  chartDir,
		Name: "curl",
		// lets hide progress bars (-s) and enable show errors (-S)
		Args: []string{"--fail", "-sS", "-u", userSecret, "--data-binary", "@" + tarFile, url},
	}, nil
}

func (o *Options) findChartRepositoryUserPassword() (string, error) {
	userName := o.RepositoryUsername
	password := o.RepositoryPassword
	if userName == "" || password == "" {
		// lets try load them from the secret directly
		client := o.KubeClient
		ns := o.Namespace
		secret, err := client.CoreV1().Secrets(ns).Get(context.TODO(), kube.SecretJenkinsChartMuseum, metav1.GetOptions{})
		if err != nil {
			secret, err = client.CoreV1().Secrets(ns).Get(context.TODO(), kube.SecretBucketRepo, metav1.GetOptions{})
		}
		if err != nil {
			log.Logger().Warnf("Could not load Secret %s or %s in namespace %s: %s", kube.SecretJenkinsChartMuseum, kube.SecretBucketRepo, ns, err)
		} else {
			if secret != nil && secret.Data != nil {
				if userName == "" {
					userName = string(secret.Data["BASIC_AUTH_USER"])
				}
				if password == "" {
					password = string(secret.Data["BASIC_AUTH_PASS"])
				}
			}
		}
	}
	if userName == "" {
		return "", fmt.Errorf("No environment variable $JX_REPOSITORY_USERNAME defined")
	}
	if password == "" {
		return "", fmt.Errorf("No environment variable $JX_REPOSITORY_PASSWORD defined")
	}
	return userName + ":" + password, nil
}
