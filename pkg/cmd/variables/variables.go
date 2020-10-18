package variables

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	v1 "github.com/jenkins-x/jx-api/v3/pkg/apis/jenkins.io/v1"
	jxc "github.com/jenkins-x/jx-api/v3/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-api/v3/pkg/config"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-gitops/pkg/variablefinders"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/activities"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/naming"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Lazily creates a .jx/variables.sh script with common pipeline environment variables
`)

	cmdExample = templates.Examples(`
		# lazily create the .jx/variables.sh file
		%s variables
	`)
)

// Options the options for the command
type Options struct {
	scmhelpers.Options
	File           string
	RepositoryName string
	RepositoryURL  string
	ConfigMapName  string
	Namespace      string
	VersionFile    string
	BuildNumber    string
	BuildID        string
	KubeClient     kubernetes.Interface
	JXClient       jxc.Interface
	Requirements   *config.RequirementsConfig
	ConfigMapData  map[string]string
	entries        map[string]*Entry
	factories      []Factory
}

// Entry a variable entry in the file on load
type Entry struct {
	Name  string
	Value string
	Index int
}

// Factory used to create a variable if its not defined locally
type Factory struct {
	Name     string
	Function func() (string, error)
}

// NewCmdVariables creates a command object for the command
func NewCmdVariables() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "variables",
		Short:   "Lazily creates a .jx/variables.sh script with common pipeline environment variables",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.File, "file", "f", filepath.Join(".jx", "variables.sh"), "the default variables file to lazily create or enrich")
	cmd.Flags().StringVarP(&o.RepositoryName, "repo-name", "n", "release-repo", "the name of the helm chart to release to. If not specified uses JX_CHART_REPOSITORY environment variable")
	cmd.Flags().StringVarP(&o.RepositoryURL, "repo-url", "u", "", "the URL to release to")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", "", "the namespace to look for the dev Environment. Defaults to the current namespace")
	cmd.Flags().StringVarP(&o.BuildNumber, "build-number", "", "", "the build number to use. If not specified defaults to $BUILD_NUMBER")
	cmd.Flags().StringVarP(&o.ConfigMapName, "configmap", "", "jenkins-x-docker-registry", "the ConfigMap used to load environment variables")
	cmd.Flags().StringVarP(&o.VersionFile, "version-file", "", "", "the file to load the version from if not specified directly or via a $VERSION environment variable. Defaults to VERSION in the current dir")
	o.Options.AddFlags(cmd)
	return cmd, o
}

// Run implements the command
func (o *Options) Validate() error {
	err := o.Options.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate scm options")
	}
	if o.VersionFile == "" {
		o.VersionFile = filepath.Join(o.Dir, "VERSION")
	}
	if o.entries == nil {
		o.entries = map[string]*Entry{}
	}
	o.JXClient, o.Namespace, err = jxclient.LazyCreateJXClientAndNamespace(o.JXClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to create jx client")
	}
	o.KubeClient, err = kube.LazyCreateKubeClient(o.KubeClient)
	if err != nil {
		return errors.Wrapf(err, "failed to create kube client")
	}

	if o.GitClient == nil {
		o.GitClient = cli.NewCLIClient("", o.CommandRunner)
	}
	o.Requirements, err = variablefinders.FindRequirements(o.JXClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to load requirements")
	}

	if o.ConfigMapData == nil {
		cm, err := o.KubeClient.CoreV1().ConfigMaps(o.Namespace).Get(context.TODO(), o.ConfigMapName, metav1.GetOptions{})
		if err != nil {
			if !apierrors.IsNotFound(err) {
				return errors.Wrapf(err, "failed to load ConfigMap %s in namespace %s", o.ConfigMapName, o.Namespace)
			}
		}
		if o.ConfigMapData == nil {
			o.ConfigMapData = map[string]string{}
		}
		if cm != nil && cm.Data != nil {
			for k, v := range cm.Data {
				name := configMapKeyToEnvVar(k)
				o.ConfigMapData[name] = v
			}
		}
	}

	if o.RepositoryURL == "" {
		o.RepositoryURL, err = variablefinders.FindRepositoryURL(o.JXClient, o.Namespace, o.Requirements)
		if err != nil {
			return errors.Wrapf(err, "failed to find chart repository URL")
		}
	}

	o.BuildNumber, err = o.GetBuildNumber()
	if err != nil {
		return errors.Wrapf(err, "failed to find build number")
	}

	o.factories = []Factory{
		{
			Name: "APP_NAME",
			Function: func() (string, error) {
				return o.Options.Repository, nil
			},
		},
		{
			Name: "BRANCH_NAME",
			Function: func() (string, error) {
				return o.Options.Branch, nil
			},
		},
		{
			Name: "BUILD_NUMBER",
			Function: func() (string, error) {
				return o.BuildNumber, nil
			},
		},
		{
			Name: "DOCKERFILE_PATH",
			Function: func() (string, error) {
				return o.FindDockerfilePath()
			},
		},
		{
			Name: "DOCKER_REGISTRY",
			Function: func() (string, error) {
				return o.dockerRegistry()
			},
		},
		{
			Name: "DOCKER_REGISTRY_ORG",
			Function: func() (string, error) {
				return o.dockerRegistryOrg()
			},
		},
		{
			Name: "JX_CHART_REPOSITORY",
			Function: func() (string, error) {
				return variablefinders.FindRepositoryURL(o.JXClient, o.Namespace, o.Requirements)
			},
		},
		{
			Name: "PIPELINE_KIND",
			Function: func() (string, error) {
				return variablefinders.FindPipelineKind(o.Branch)
			},
		},
		{
			Name: "REPO_NAME",
			Function: func() (string, error) {
				return o.Options.Repository, nil
			},
		},
		{
			Name: "REPO_OWNER",
			Function: func() (string, error) {
				return o.Options.Owner, nil
			},
		},
		{
			Name: "VERSION",
			Function: func() (string, error) {
				return variablefinders.FindVersion(o.VersionFile, o.Options.Branch, o.BuildNumber)
			},
		},
	}
	return nil
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate")
	}

	file := o.File
	if o.Dir != "" {
		file = filepath.Join(o.Dir, file)
	}
	exists, err := files.FileExists(file)
	if err != nil {
		return errors.Wrapf(err, "failed to check if file exists %s", file)
	}
	source := ""

	if exists {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			return errors.Wrapf(err, "failed to read file %s", file)
		}
		source = string(data)
		lines := strings.Split(source, "\n")
		for i, line := range lines {
			if strings.HasSuffix(line, "export ") {
				text := strings.TrimSpace(line[len("export "):])
				idx := strings.Index(text, "=")
				if idx > 0 {
					name := strings.TrimSpace(text[0:idx])
					if name != "" {
						value := strings.TrimSpace(text[idx+1:])

						entry := &Entry{
							Name:  name,
							Value: value,
							Index: i,
						}
						o.entries[name] = entry
					}
				}
			}
		}

		source += "\n\n"
	}

	buf := strings.Builder{}

	for _, f := range o.factories {
		name := f.Name
		entry := o.entries[name]
		value := ""
		if entry != nil {
			value = entry.Value
		}

		if value == "" {
			if f.Function == nil {
				return errors.Errorf("no function for variable %s", name)
			}
			value, err = f.Function()
			if err != nil {
				return errors.Wrapf(err, "failed to evaluate function for variable %s", name)
			}
			if value != "" {
				log.Logger().Infof("export %s=\"%s\"", name, value)

				line := fmt.Sprintf("export %s=\"%s\"", name, value)

				if buf.Len() == 0 {
					buf.WriteString("\n# generated by: jx gitops variables\n")
				}
				buf.WriteString(line)
				buf.WriteString("\n")
			}
		}
	}
	text := buf.String()
	if text == "" {
		log.Logger().Infof("no new variables added to %s", info(file))
		return nil
	}
	source += text
	dir := filepath.Dir(file)
	err = os.MkdirAll(dir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to create dir %s", dir)
	}
	err = ioutil.WriteFile(file, []byte(source), files.DefaultFileWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save %s", file)
	}
	log.Logger().Infof("added variables to file: %s", info(file))

	_, err = gitclient.AddAndCommitFiles(o.GitClient, o.Dir, "added variables")
	if err != nil {
		return errors.Wrapf(err, "failed to commit changes")
	}
	return nil
}

func (o *Options) dockerRegistry() (string, error) {
	answer := ""
	if o.Requirements != nil {
		answer = o.Requirements.Cluster.Registry
	}
	if answer == "" {
		answer = o.ConfigMapData["DOCKER_REGISTRY"]
	}
	return answer, nil
}

func (o *Options) dockerRegistryOrg() (string, error) {
	answer := ""
	if answer == "" {
		answer = o.ConfigMapData["DOCKER_REGISTRY_ORG"]
	}
	if answer == "" {
		if o.Requirements != nil {
			answer = o.Requirements.Cluster.DockerRegistryOrg
			if answer == "" && o.Requirements.Cluster.Provider == "gke" {
				answer = o.Requirements.Cluster.ProjectID
			}
		}
	}
	if answer == "" {
		answer = naming.ToValidName(o.Options.Owner)
	}
	return answer, nil
}

// GetBuildNumber returns the build number from BUILD_NUMBER or uses PipelineActivities to create/find it
func (o *Options) GetBuildNumber() (string, error) {
	if o.BuildNumber == "" {
		o.BuildNumber = os.Getenv("BUILD_NUMBER")
		if o.BuildNumber == "" {
			var err error
			buildID := o.GetBuildID()
			if buildID != "" {
				o.BuildNumber, err = o.FindBuildNumber(buildID)
				if err != nil {
					return "", errors.Wrapf(err, "failed to find BuildNumber")
				}
			} else {
				log.Logger().Warnf("no $BUILD_ID found so cannot create the BUILD_NUMBER")
			}
		}
	}
	return o.BuildNumber, nil
}

// FindBuildNumber finds the build number for the given build ID
func (o *Options) FindBuildNumber(buildID string) (string, error) {
	// lets try find a PipelineActivity with this build ID...
	activityInterface := o.JXClient.JenkinsV1().PipelineActivities(o.Namespace)

	owner := o.Options.Owner
	repository := o.Options.Repository
	branch := o.Options.Branch
	var activitySlice []v1.PipelineActivity

	selectors := []string{
		"owner=" + naming.ToValidName(owner) +
			",repository=" + naming.ToValidName(repository) +
			",branch=" + branch,
		"lighthouse.jenkins-x.io/refs.org=" + naming.ToValidName(owner) +
			",lighthouse.jenkins-x.io/refs.repo=" + naming.ToValidName(repository) +
			",lighthouse.jenkins-x.io/branch=" + branch,
	}
	for _, selector := range selectors {
		resources, err := activityInterface.List(context.TODO(), metav1.ListOptions{
			LabelSelector: selector,
		})
		if err != nil && !apierrors.IsNotFound(err) {
			return "", errors.Wrapf(err, "failed to find PipelineActivity resources in namespace %s with selector %s", o.Namespace, selector)
		}
		if resources != nil {
			activitySlice = append(activitySlice, resources.Items...)
		}
	}

	maxBuild := 0
	for i := range activitySlice {
		pa := &activitySlice[i]
		labels := pa.Labels
		if labels == nil {
			continue
		}
		if labels["buildID"] == buildID || labels["lighthouse.jenkins-x.io/buildNum"] == buildID {
			if pa.Spec.Build == "" {
				log.Logger().Warnf("PipelineActivity %s does not have a spec.build value", pa.Name)
			} else {
				return pa.Spec.Build, nil
			}
			continue
		}
		if pa.Spec.Build != "" {
			i, err := strconv.Atoi(pa.Spec.Build)
			if err != nil {
				log.Logger().Warnf("PipelineActivity %s has an invalid spec.build number %s should be an integer: %s", pa.Name, pa.Spec.Build, err.Error())
			} else {
				if i > maxBuild {
					maxBuild = i
				}
			}
		}
	}
	o.BuildNumber = strconv.Itoa(maxBuild + 1)

	// lets lazy create a new PipelineActivity for this new build number...
	pipeline := fmt.Sprintf("%s/%s/%s", owner, repository, branch)
	name := naming.ToValidName(pipeline + "-" + o.BuildNumber)

	key := &activities.PromoteStepActivityKey{
		PipelineActivityKey: activities.PipelineActivityKey{
			Name:     name,
			Pipeline: pipeline,
			Build:    o.BuildNumber,
			GitInfo: &giturl.GitRepository{
				Name:         repository,
				Organisation: owner,
			},
			Labels: map[string]string{
				"buildID": buildID,
			},
		},
	}
	_, _, err := key.GetOrCreate(o.JXClient, o.Namespace)
	if err != nil {
		return o.BuildNumber, errors.Wrapf(err, "failed to lazily create PipelineActivity %s", name)
	}
	return o.BuildNumber, nil
}

// GetBuildID returns the current build ID
func (o *Options) GetBuildID() string {
	if o.BuildID == "" {
		o.BuildID = os.Getenv("BUILD_ID")
	}
	return o.BuildID
}

// FindDockerfilePath finds the dockerfile path to use relative to the current directory
func (o *Options) FindDockerfilePath() (string, error) {
	kind, err := variablefinders.FindPipelineKind(o.Branch)
	if err != nil {
		return "", errors.Wrapf(err, "failed to find pipeline kind")
	}
	if kind == "pullrequest" {
		name := "Dockerfile-preview"
		path := filepath.Join(o.Dir, name)
		exists, err := files.FileExists(path)
		if err != nil {
			return "", errors.Wrapf(err, "failed to detect file %s", path)
		}
		if exists {
			return name, nil
		}
	}
	return "Dockerfile", nil

}

func configMapKeyToEnvVar(k string) string {
	text := strings.ToUpper(k)
	text = strings.ReplaceAll(text, ".", "_")
	text = strings.ReplaceAll(text, "-", "_")
	return text
}
