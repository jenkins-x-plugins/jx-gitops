package release

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/chart"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/ghpages"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/plugins"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/variablefinders"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	jxc "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
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

	defaultReadMe = `
## Chart Repository

[Helm](https://helm.sh) must be installed to use the charts.
Please refer to Helm's [documentation](https://helm.sh/docs/) to get started.

### Searching for charts

Once Helm is set up properly, add the repo as follows:

    helm repo add myrepo %s

you can search the charts via:

    helm search repo myfilter

## View the YAML

You can have a look at the underlying charts YAML at: [index.yaml](index.yaml)
`
)

// Options the options for the command
type Options struct {
	UseHelmPlugin        bool
	NoRelease            bool
	ChartOCI             bool
	ChartPages           bool
	NoOCILogin           bool
	Artifactory          bool
	HelmBinary           string
	Dir                  string
	ChartsDir            string
	RepositoryName       string
	RepositoryURL        string
	RepositoryUsername   string
	RepositoryPassword   string
	RepositoryNested     string
	GithubPagesBranch    string
	GithubPagesURL       string
	Version              string
	VersionFile          string
	Namespace            string
	ContainerRegistryOrg string
	GitHubPagesDir       string
	IgnoreChartNames     []string
	KubeClient           kubernetes.Interface
	JXClient             jxc.Interface
	GitClient            gitclient.Interface
	CommandRunner        cmdrunner.CommandRunner
	Requirements         *jxcore.RequirementsConfig
	ReleasedCharts       int
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
	cmd.Flags().StringVarP(&o.Dir, "dir", "", ".", "the root directory to look for .jx/requirements.yaml")
	cmd.Flags().StringVarP(&o.ChartsDir, "charts-dir", "c", "charts", "the directory to look for helm charts to release")
	cmd.Flags().StringVarP(&o.RepositoryName, "repo-name", "n", "release-repo", "the name of the helm chart to release to. If not specified uses JX_CHART_REPOSITORY environment variable")
	cmd.Flags().StringVarP(&o.RepositoryURL, "repo-url", "u", "", "the URL to release to")
	cmd.Flags().StringVarP(&o.RepositoryUsername, "repo-username", "", "", "the username to access the chart repository. If not specified defaults to the environment variable $JX_REPOSITORY_USERNAME")
	cmd.Flags().StringVarP(&o.RepositoryPassword, "repo-password", "", "", "the password to access the chart repository. If not specified defaults to the environment variable $JX_REPOSITORY_PASSWORD")
	cmd.Flags().StringVarP(&o.RepositoryNested, "repo-nested", "", "", "the nested repository inside the repository. If not specified defaults to empty (not nested repo)")
	cmd.Flags().StringVarP(&o.Version, "version", "", "", "specify the version to release")
	cmd.Flags().StringVarP(&o.VersionFile, "version-file", "", "VERSION", "the file to load the version from if not specified directly or via a $VERSION environment variable")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", "", "the namespace to look for the dev Environment. Defaults to the current namespace")
	cmd.Flags().StringVarP(&o.GithubPagesBranch, "repository-branch", "", "gh-pages", "the branch used if using GitHub Pages for the helm chart")
	cmd.Flags().StringVarP(&o.GithubPagesURL, "ghpage-url", "", "", "the github pages URL used if creating the first README.md in the github pages branch so we can link to how to add a chart repository")
	cmd.Flags().StringArrayVarP(&o.IgnoreChartNames, "ignore", "I", []string{"preview"}, "the names of helm charts to not release")
	cmd.Flags().BoolVarP(&o.ChartPages, "pages", "", false, "use github pages to release charts")
	cmd.Flags().BoolVarP(&o.ChartOCI, "oci", "", false, "treat the repository as an OCI container registry. If not specified its defaulted from the cluster.chartOCI flag on the 'jx-requirements.yml' file")
	cmd.Flags().BoolVarP(&o.Artifactory, "artifactory", "", false, "use artifactory mode for publishing the chart which involves using an artifactory header and -T for pushing the chart")
	cmd.Flags().BoolVarP(&o.NoOCILogin, "no-oci-login", "", false, "disables using the 'helm registry login' command when using OCI")
	cmd.Flags().BoolVarP(&o.NoRelease, "no-release", "", false, "disables publishing the release. Useful for a Pull Request pipeline")
	cmd.Flags().BoolVarP(&o.UseHelmPlugin, "use-helm-plugin", "", false, "uses the jx binary plugin for helm rather than whatever helm is on the $PATH")
	return cmd, o
}

// Run implements the command
func (o *Options) Validate() error {
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.QuietCommandRunner
	}
	if !o.Artifactory && os.Getenv("ARTIFACTORY_CHART_REPOSITORY") == "true" {
		o.Artifactory = true
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
		if o.RepositoryUsername == "" {
			o.RepositoryUsername = os.Getenv("GIT_USERNAME")
		}
		if o.RepositoryUsername == "" {
			o.RepositoryUsername = os.Getenv("GITHUB_ACTOR")
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
	requirements, err := variablefinders.FindRequirements(o.GitClient, o.JXClient, o.Namespace, o.Dir, "", "")
	if err != nil {
		return errors.Wrapf(err, "failed to load requirements")
	}
	if requirements != nil {
		o.Requirements = requirements
		switch requirements.Cluster.ChartKind {
		case jxcore.ChartRepositoryTypeOCI:
			o.ChartOCI = true
			o.ChartPages = false
		case jxcore.ChartRepositoryTypePages:
			o.ChartOCI = false
			o.ChartPages = true
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

		if stringhelpers.StringArrayIndex(o.IgnoreChartNames, name) >= 0 {
			log.Logger().Infof("not releasing chart %s", info(name))
			continue
		}
		log.Logger().Infof("releasing chart %s", info(name))

		// find the repository URL
		if repoURL == "" {
			repoURL, err = variablefinders.FindRepositoryURL(o.Requirements, o.ContainerRegistryOrg, name)
			if err != nil {
				return errors.Wrapf(err, "failed to find chart repository URL")
			}
		}

		if o.ChartPages {
			err = o.ChartPageRegistry(repoURL, chartDir, name)
			if err != nil {
				return errors.Wrapf(err, "failed to create chart pages release in dir %s", chartDir)
			}
		} else if o.ChartOCI {
			err = o.OCIRegistry(repoURL, chartDir, name)
			if err != nil {
				return errors.Wrapf(err, "failed to create OCI chart release in dir %s", chartDir)
			}
		} else {
			err = o.BasicRegistry(repoURL, chartDir, name)
			if err != nil {
				return errors.Wrapf(err, "failed to create chart release in dir %s", chartDir)
			}
		}
		count++
	}

	log.Logger().Infof("released %d charts from the charts dir: %s", count, dir)
	o.ReleasedCharts = count
	return nil
}

func (o *Options) OCIRegistry(repoURL, chartDir, name string) error {
	qualifiedChartName := fmt.Sprintf("%s/%s:%s", repoURL, name, o.Version)
	var c *cmdrunner.Command
	var err error

	if !o.NoOCILogin {
		c = &cmdrunner.Command{
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

func (o *Options) ChartPageRegistry(repoURL, chartDir, name string) error {
	err := o.BuildAndPackage(chartDir)
	if err != nil {
		return errors.Wrapf(err, "failed to package chart")
	}

	if o.NoRelease {
		log.Logger().Infof("disabling the chart publish")
		return nil
	}

	if o.GitHubPagesDir == "" || o.GithubPagesURL == "" {
		if repoURL == "" || o.RepositoryPassword == "" || o.GithubPagesURL == "" {
			discover := &scmhelpers.Options{
				Dir:             ".",
				JXClient:        o.JXClient,
				GitClient:       o.GitClient,
				CommandRunner:   o.CommandRunner,
				DiscoverFromGit: true,
			}
			err = discover.Validate()
			if err != nil {
				return errors.Wrapf(err, "failed to discover git repository")
			}

			if repoURL == "" {
				repoURL = discover.SourceURL
			}
			if o.RepositoryPassword == "" {
				o.RepositoryPassword = discover.GitToken
			}
			if o.RepositoryUsername == "" {
				o.RepositoryUsername = discover.ScmClient.Username
			}
			if o.GithubPagesURL == "" {
				o.GithubPagesURL = fmt.Sprintf("https://%s.github.io/%s/", discover.Owner, discover.Repository)
			}
		}
		if repoURL == "" {
			return options.MissingOption("repo-url")
		}
		if o.RepositoryUsername == "" {
			return options.MissingOption("repo-username")
		}
		if o.RepositoryPassword == "" {
			return options.MissingOption("repo-password")
		}
		if o.GithubPagesBranch == "" {
			o.GithubPagesBranch = "gh-pages"
		}

		o.GitHubPagesDir, err = ghpages.CloneGitHubPagesToDir(o.GitClient, repoURL, o.GithubPagesBranch, o.RepositoryUsername, o.RepositoryPassword)
		if err != nil {
			return errors.Wrapf(err, "failed to clone the github pages repo %s branch %s", repoURL, o.GithubPagesBranch)
		}

		if o.GitHubPagesDir == "" {
			return errors.Errorf("no github pages clone dir")
		}
	}

	// lets copy files
	fs, err := ioutil.ReadDir(chartDir)
	if err != nil {
		return errors.Wrapf(err, "failed to read chart dir %s", chartDir)
	}
	for _, f := range fs {
		name := f.Name()
		if f.IsDir() || !strings.HasSuffix(name, ".tgz") {
			continue
		}
		path := filepath.Join(chartDir, name)
		tofile := filepath.Join(o.GitHubPagesDir, name)

		err = files.CopyFile(path, tofile)
		if err != nil {
			return errors.Wrapf(err, "failed to copy %s to %s", path, tofile)
		}
	}

	// lets re-index
	c := &cmdrunner.Command{
		Dir:  o.GitHubPagesDir,
		Name: o.HelmBinary,
		Args: []string{"repo", "index", "."},
	}
	_, err = o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to index helm repository")
	}

	// lets add a README if its missing
	readmePath := filepath.Join(o.GitHubPagesDir, "README.md")
	exists, err := files.FileExists(readmePath)
	if err != nil {
		return errors.Wrapf(err, "failed to check for file %s", readmePath)
	}
	if !exists {
		readmeText := fmt.Sprintf(defaultReadMe, o.GithubPagesURL)
		err = ioutil.WriteFile(readmePath, []byte(readmeText), files.DefaultFileWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to save %s", readmePath)
		}
	}
	_, err = gitclient.AddAndCommitFiles(o.GitClient, o.GitHubPagesDir, "chore: add helm chart")
	if err != nil {
		return errors.Wrapf(err, "failed to add helm chart to git")
	}
	log.Logger().Infof("added helm charts to github pages repository %s", repoURL)
	_, err = o.GitClient.Command(o.GitHubPagesDir, "push", "--set-upstream", "origin", o.GithubPagesBranch)
	if err != nil {
		return errors.Wrapf(err, "failed to push changes")
	}
	log.Logger().Infof("pushd github pages to %s", repoURL)
	return nil
}

// Setup sets up the storage in the given directory
func (o *Options) GitCloneGitHubPages(repoURL, branch string) (string, error) {
	return ghpages.CloneGitHubPagesToDir(o.GitClient, repoURL, branch, o.RepositoryUsername, o.RepositoryPassword)
}

func (o *Options) BasicRegistry(repoURL, chartDir, name string) error {
	username, password, err := o.findChartRepositoryUserPassword()
	if err != nil {
		return errors.Wrapf(err, "failed to find chart repository user and password")
	}

	c := &cmdrunner.Command{
		Dir:  chartDir,
		Name: o.HelmBinary,
		Args: []string{"repo", "add", "--username", username, "--password", password, o.RepositoryName, repoURL},
	}
	_, err = o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to add remote repo")
	}

	err = o.BuildAndPackage(chartDir)
	if err != nil {
		return errors.Wrapf(err, "failed to package chart")
	}

	if o.NoRelease {
		log.Logger().Infof("disabling the chart publish")
		return nil
	}

	c = o.createPublishCommand(repoURL, name, chartDir, username, password)

	_, err = o.CommandRunner(c)
	if err != nil {
		return errors.Wrapf(err, "failed to publish")
	}
	return nil
}

func (o *Options) BuildAndPackage(chartDir string) error {
	chartFile := filepath.Join(chartDir, "Chart.yaml")
	exists, err := files.FileExists(chartFile)
	if err != nil {
		return errors.Wrapf(err, "failed to check file exists %s", chartFile)
	}

	chartDef := &chart.Chart{}
	if exists {
		err = yamls.LoadFile(chartFile, chartDef)
		if err != nil {
			return errors.Wrapf(err, "failed to load Chart.yaml")
		}

		for i, dependency := range chartDef.Dependencies {
			log.Logger().Infof("Adding repository for dependency %s", dependency.Name)
			if dependency.Repository != "" {
				c := &cmdrunner.Command{
					Dir:  chartDir,
					Name: o.HelmBinary,
					Args: []string{"repo", "add", strconv.Itoa(i), dependency.Repository},
				}
				_, err = o.CommandRunner(c)
				if err != nil {
					return errors.Wrapf(err, "failed to add repository")
				}
			} else {
				log.Logger().Infof("Skipping local dependency %s", dependency.Name)
			}
		}
	}

	c := &cmdrunner.Command{
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
	return nil
}

func (o *Options) createPublishCommand(repoURL, name, chartDir, username, password string) *cmdrunner.Command {
	tarFile := name + "-" + o.Version + ".tgz"

	if strings.HasPrefix(repoURL, "gs:") {
		// use gcs to push the chart
		return &cmdrunner.Command{
			Dir:  chartDir,
			Name: o.HelmBinary,
			Args: []string{"gcs", "push", tarFile, o.RepositoryName},
		}
	}

	if strings.HasPrefix(repoURL, "s3:") {
		// use s3 to push the chart
		return &cmdrunner.Command{
			Dir:  chartDir,
			Name: o.HelmBinary,
			Args: []string{"s3", "push", tarFile, o.RepositoryName},
		}
	}

	if o.Artifactory {
		// lets try detect the git repository name
		url := stringhelpers.UrlJoin(repoURL, tarFile)

		repoName := os.Getenv("REPO_NAME")
		if repoName != "" {
			url = stringhelpers.UrlJoin(repoURL, repoName, tarFile)
		}

		apiKey := "X-JFrog-Art-Api:" + password

		return &cmdrunner.Command{
			Dir:  chartDir,
			Name: "curl",
			// lets hide progress bars (-s) and enable show errors (-S)
			Args: []string{"--fail", "-sS", "-H", apiKey, "-T", tarFile, url},
		}
	}
	userSecret := username + ":" + password

	url := stringhelpers.UrlJoin(repoURL, "/api/charts")

	if o.RepositoryNested != "" {
		url = stringhelpers.UrlJoin(repoURL, "/api/", o.RepositoryNested, "/charts")
	}

	return &cmdrunner.Command{
		Dir:  chartDir,
		Name: "curl",
		// lets hide progress bars (-s) and enable show errors (-S)
		Args: []string{"--fail", "-sS", "-u", userSecret, "--data-binary", "@" + tarFile, url},
	}
}

func (o *Options) findChartRepositoryUserPassword() (string, string, error) {
	userName := o.RepositoryUsername
	password := o.RepositoryPassword
	if userName == "" || password == "" {
		// lets try load them from the secret directly
		client := o.KubeClient
		ns := o.Namespace
		var secret *corev1.Secret
		var err error
		secretName := kube.SecretJenkinsChartMuseum
		useBucketRepo := true
		if o.Requirements != nil && o.Requirements.Cluster.ChartSecret != "" {
			secretName = o.Requirements.Cluster.ChartSecret
			useBucketRepo = false
		}
		secret, err = client.CoreV1().Secrets(ns).Get(context.TODO(), secretName, metav1.GetOptions{})
		if err != nil && useBucketRepo {
			secret, err = client.CoreV1().Secrets(ns).Get(context.TODO(), kube.SecretBucketRepo, metav1.GetOptions{})
		}
		if err != nil {
			log.Logger().Warnf("Could not load Secret %s or %s in namespace %s: %s", secretName, kube.SecretBucketRepo, ns, err)
		} else if secret != nil && secret.Data != nil {
			if password == "" {
				password = string(secret.Data["BASIC_AUTH_PASS"])
				if password == "" {
					password = string(secret.Data["password"])
				}
			}
			if userName == "" {
				userName = string(secret.Data["BASIC_AUTH_USER"])
				if userName == "" {
					userName = string(secret.Data["username"])
					if userName == "" && password != "" {
						// for easier integration with nexus lets default to admin
						userName = "admin"
					}
				}
			}

		}
	}
	if userName == "" {
		return "", "", fmt.Errorf("no environment variable $JX_REPOSITORY_USERNAME defined")
	}
	if password == "" {
		return "", "", fmt.Errorf("no environment variable $JX_REPOSITORY_PASSWORD defined")
	}
	return userName, password, nil
}
