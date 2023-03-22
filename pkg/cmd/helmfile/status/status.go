package status

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/releasereport"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/sourceconfigs"
	"github.com/jenkins-x/go-scm/scm"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/v3/pkg/requirements"
	"github.com/jenkins-x/jx-helpers/v3/pkg/scmhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	info = termcolor.ColorInfo

	statusLong = templates.LongDesc(`
		Updates the git deployment status after a release
`)

	statusExample = templates.Examples(`
		# update the status in git after a release
		%s helmfile status
	`)

	titleCaser = cases.Title(language.English)
)

// Options the options for viewing running PRs
type Options struct {
	scmhelpers.Factory
	Dir               string
	FailOnError       bool
	AutoInactive      bool
	SourceConfig      *v1alpha1.SourceConfig
	NamespaceReleases []*releasereport.NamespaceReleases
	Requirements      *jxcore.Requirements
	TestGitToken      string
	EnvironmentNames  map[string]string
	EnvironmentURLs   map[string]string
}

// NewCmdHelmfileStatus creates a command object for the command
func NewCmdHelmfileStatus() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "status",
		Short:   "Updates the git deployment status after a release",
		Long:    statusLong,
		Example: fmt.Sprintf(statusExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory that contains the content")
	cmd.Flags().BoolVarP(&o.FailOnError, "fail", "f", false, "if enabled then fail the boot pipeline if we cannot report the deployment status")
	cmd.Flags().BoolVarP(&o.AutoInactive, "auto-inactive", "a", true, "if enabled then the the status of previous deployments will be set to inactive (Default: true)")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	path := filepath.Join(o.Dir, "docs", "releases.yaml")
	exists, err := files.FileExists(path)
	if err != nil {
		return errors.Wrapf(err, "failed to check file exists %s", path)
	}
	if !exists {
		log.Logger().Infof("no report at file %s so cannot report deployment status", info(path))
		return nil
	}

	err = yamls.LoadFile(path, &o.NamespaceReleases)
	if err != nil {
		return errors.Wrapf(err, "failed to load %s", path)
	}

	o.Requirements, _, err = jxcore.LoadRequirementsConfig(o.Dir, false)
	if err != nil {
		return errors.Wrapf(err, "failed to load requirements in dir %s", o.Dir)
	}
	if o.EnvironmentNames == nil {
		o.EnvironmentNames = map[string]string{}
	}
	if o.EnvironmentURLs == nil {
		o.EnvironmentURLs = map[string]string{}
	}
	for k := range o.Requirements.Spec.Environments {
		e := o.Requirements.Spec.Environments[k]
		ns := e.Namespace
		if ns == "" {
			ns = "jx"
			if e.Key != "dev" {
				ns = "jx-" + e.Key
			}
		}
		o.EnvironmentNames[ns] = titleCaser.String(e.Key)

		envURL := requirements.EnvironmentGitURL(&o.Requirements.Spec, e.Key)
		o.EnvironmentURLs[ns] = envURL
		if e.Key == "dev" {
			o.EnvironmentURLs["dev"] = envURL
		}
	}

	o.SourceConfig, err = sourceconfigs.LoadSourceConfig(o.Dir, false)
	if err != nil {
		return errors.Wrapf(err, "failed to load source config from dir %s", o.Dir)
	}
	if len(o.SourceConfig.Spec.Groups) == 0 {
		log.Logger().Warnf("no source config found in dir %s. Will assume all repos are in the current organisation as gitops repo", o.Dir)
		ctx := context.Background()

		c := o.Requirements.Spec.Cluster
		gitServer := stringhelpers.FirstNotEmptyString(c.GitServer, giturl.GitHubURL)
		twoHoursAgo := time.Now().Add(-2 * time.Hour)
		for _, nsr := range o.NamespaceReleases {
			for _, release := range nsr.Releases {
				if twoHoursAgo.Before(release.LastDeployed.Time) {
					continue
				}

				env := o.getEnvForNamespace(nsr.Namespace)

				err = o.CreateNewScmClient(c.EnvironmentGitOwner, gitServer, c.GitKind)
				if err != nil {
					return errors.Wrapf(err, "failed to create scm client for owner %s", c.EnvironmentGitOwner)
				}

				err = o.updateStatus(ctx, env, gitServer, c.EnvironmentGitOwner, release.Name, release)
				if err != nil {
					if o.FailOnError {
						return errors.Wrapf(err, "failed to update status for repository %s/%s", c.EnvironmentGitOwner, release.Name)
					}
					log.Logger().Warnf("failed to update status for repository %s/%s : %s", c.EnvironmentGitOwner, release.Name, err.Error())
				}
			}
		}
	} else {
		for i := range o.SourceConfig.Spec.Groups {
			group := &o.SourceConfig.Spec.Groups[i]
			for j := range group.Repositories {
				repo := &group.Repositories[j]
				err = sourceconfigs.DefaultValues(o.SourceConfig, group, repo)
				if err != nil {
					return errors.Wrapf(err, "failed to default SourceConfig")
				}

				err = o.updateStatuses(group, repo)
				if err != nil {
					if o.FailOnError {
						return errors.Wrapf(err, "failed to update status for repository %s/%s", group.Owner, repo.Name)
					}
					log.Logger().Warnf("failed to update status for repository %s/%s : %s", group.Owner, repo.Name, err.Error())
				}
			}
		}
	}
	return nil
}

func (o *Options) getEnvForNamespace(ns string) *environment {
	env := &environment{
		name: o.EnvironmentNames[ns],
		url:  o.EnvironmentURLs[ns],
	}

	if env.name == "" {
		env.name = titleCaser.String(strings.TrimPrefix(ns, "jx-"))
	}
	if env.url == "" {
		env.url = o.EnvironmentURLs["dev"]
	}
	return env
}

func (o *Options) updateStatuses(group *v1alpha1.RepositoryGroup, repo *v1alpha1.Repository) error {
	ctx := context.Background()
	twoHoursAgo := time.Now().Add(-2 * time.Hour)
	for _, nsr := range o.NamespaceReleases {
		for _, release := range nsr.Releases {
			// TODO could use source of the release to match on to reduce name clashes?
			if release.Name != repo.Name {
				continue
			}
			if twoHoursAgo.Before(release.LastDeployed.Time) {
				continue
			}

			env := o.getEnvForNamespace(nsr.Namespace)

			err := o.CreateNewScmClient(group.Owner, group.Provider, group.ProviderKind)
			if err != nil {
				return errors.Wrapf(err, "failed to create scm client for repository %s/%s", group.Owner, repo.Name)
			}

			err = o.updateStatus(ctx, env, group.Provider, group.Owner, repo.Name, release)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

type environment struct {
	name string
	url  string
}

func (o *Options) updateStatus(ctx context.Context, env *environment, provider, owner, repoName string, release *releasereport.ReleaseInfo) error {
	if release.Version == "" {
		log.Logger().Warnf("missing version for release %s in environment %s", repoName, env.name)
		return nil
	}

	fullRepoName := scm.Join(owner, repoName)
	if o.ScmClient.Deployments == nil {
		log.Logger().Warnf("cannot update deployment status of release %s as the git server %s does not support Deployments", fullRepoName, provider)
		return nil
	}

	deployment, err := o.FindExistingDeploymentInEnvironment(ctx, fullRepoName, env.name)
	if err != nil {
		return err
	}

	ref := "v" + release.Version

	if deployment == nil {
		deployment, err = o.CreateNewDeployment(ctx, ref, env.name, fullRepoName)
		if err != nil {
			return err
		}

	} else if ref == deployment.Ref {
		// We should ignore releases that are the same as the current deployment
		log.Logger().Infof("existing deployment for %s is the same version as release (%s). Skipping deployment", fullRepoName, ref)
		return nil
	}

	deploymentStatusInput := &scm.DeploymentStatusInput{
		State:           "success",
		TargetLink:      release.ApplicationURL,
		LogLink:         release.LogsURL,
		Description:     fmt.Sprintf("Deployment %s", strings.TrimPrefix(release.Version, "v")),
		Environment:     env.name,
		EnvironmentLink: env.url,
		AutoInactive:    o.AutoInactive,
	}

	status, _, err := o.ScmClient.Deployments.CreateStatus(ctx, fullRepoName, deployment.ID, deploymentStatusInput)
	if err != nil {
		return errors.Wrapf(err, "failed to create DeploymentStatus for repository %s and ref %s", fullRepoName, ref)
	}
	log.Logger().Infof("created DeploymentStatus for repository %s ref %s at %s with Logs URL %s and Target URL %s", fullRepoName, ref, status.ID, release.LogsURL, release.ApplicationURL)
	return nil
}

func (o *Options) CreateNewDeployment(ctx context.Context, ref, environment, fullRepoName string) (*scm.Deployment, error) {
	_, name := scm.Split(fullRepoName)
	deploymentInput := &scm.DeploymentInput{
		Ref:                   ref,
		Task:                  "deploy",
		Environment:           environment,
		Description:           fmt.Sprintf("release %s for version %s", name, strings.TrimPrefix(ref, "v")),
		RequiredContexts:      nil,
		AutoMerge:             false,
		TransientEnvironment:  false,
		ProductionEnvironment: strings.Contains(strings.ToLower(environment), "prod"),
	}

	deployment, _, err := o.ScmClient.Deployments.Create(ctx, fullRepoName, deploymentInput)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to create Deployment for repository %s and ref %s", fullRepoName, ref)
	}
	log.Logger().Infof("created Deployment for release %s at %s", fullRepoName, deployment.Link)
	return deployment, nil
}

func (o *Options) FindExistingDeploymentInEnvironment(ctx context.Context, fullRepoName, environment string) (*scm.Deployment, error) {
	_, name := scm.Split(fullRepoName)
	// lets try find the existing deployment if it exists
	deployments, _, err := o.ScmClient.Deployments.List(ctx, fullRepoName, scm.ListOptions{})
	if err != nil && !scmhelpers.IsScmNotFound(err) {
		return nil, err
	}
	for _, d := range deployments {
		if d.Name == name && d.Environment == environment {
			log.Logger().Infof("found existing deployment %s", d.Link)
			return d, nil
		}
	}
	return nil, nil
}

func (o *Options) CreateNewScmClient(owner, server, gitKind string) error {
	if server == "" {
		return errors.Errorf("no provider defined for owner %s", owner)
	}
	if gitKind == "" {
		gitKind = giturl.SaasGitKind(server)
	}
	if gitKind == "" {
		return errors.Errorf("no git provider kind for owner %s", owner)
	}

	// lets find the credentials from git...
	o.Factory = scmhelpers.Factory{
		GitKind:      gitKind,
		GitServerURL: server,
		GitToken:     o.TestGitToken,
	}
	_, err := o.Create()
	return err
}
