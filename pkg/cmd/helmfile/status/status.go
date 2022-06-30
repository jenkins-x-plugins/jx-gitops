package status

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

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
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
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
		// ToDo: Replace once we upgrade to go1.18
		o.EnvironmentNames[ns] = strings.Title(e.Key) //nolint:staticcheck

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
	if o.SourceConfig == nil {
		return errors.Errorf("no source config found in dir %s", o.Dir)
	}

	for i := range o.SourceConfig.Spec.Groups {
		group := &o.SourceConfig.Spec.Groups[i]
		for j := range group.Repositories {
			repo := &group.Repositories[j]
			err = sourceconfigs.DefaultValues(o.SourceConfig, group, repo)
			if err != nil {
				return errors.Wrapf(err, "failed to default SourceConfig")
			}

			err = o.updateStatus(group, repo)
			if err != nil {
				if o.FailOnError {
					return errors.Wrapf(err, "failed to update status for repository %s/%s", group.Owner, repo.Name)
				}
				log.Logger().Warnf("failed to update status for repository %s/%s : %s", group.Owner, repo.Name, err.Error())
			}
		}
	}
	return nil
}

func (o *Options) updateStatus(group *v1alpha1.RepositoryGroup, repo *v1alpha1.Repository) error {
	ctx := context.Background()

	owner := group.Owner
	server := group.Provider

	repoName := repo.Name
	fullName := scm.Join(owner, repoName)

	for _, nsr := range o.NamespaceReleases {
		for _, release := range nsr.Releases {

			// TODO could use source of the release to match on to reduce name clashes?
			if release.Name != repo.Name {
				continue
			}

			version := release.Version
			releaseNS := nsr.Namespace
			environment := o.EnvironmentNames[releaseNS]
			if environment == "" {
				environment = releaseNS
			}

			appName := repoName
			targetLink := release.ApplicationURL
			logLink := release.LogsURL
			description := fmt.Sprintf("Deployment %s", strings.TrimPrefix(version, "v"))

			environmentLink := o.EnvironmentURLs[releaseNS]
			if environmentLink == "" {
				environmentLink = o.EnvironmentURLs["dev"]
			}

			if version == "" {
				log.Logger().Warnf("missing version for release %s in environment %s", appName, environment)
				continue
			}
			ref := "v" + version

			scmClient, err := o.CreateScmClient(group, repo)
			if err != nil {
				return errors.Wrapf(err, "failed to create scm client for repository %s/%s", owner, repoName)
			}

			if scmClient.Deployments == nil {
				log.Logger().Warnf("cannot update deployment status of release %s as the git server %s does not support Deployments", fullName, server)
				return nil
			}

			// lets try find the existing deployment if it exists
			deployments, _, err := scmClient.Deployments.List(ctx, fullName, scm.ListOptions{})
			if err != nil && !scmhelpers.IsScmNotFound(err) {
				return err
			}
			var deployment *scm.Deployment
			for _, d := range deployments {
				if d.Ref == ref && d.Environment == environment {
					log.Logger().Infof("found existing deployment %s", d.Link)
					deployment = d
					break
				}
			}

			if deployment == nil {
				deploymentInput := &scm.DeploymentInput{
					Ref:                   ref,
					Task:                  "deploy",
					Environment:           environment,
					Description:           fmt.Sprintf("release %s for version %s", appName, version),
					RequiredContexts:      nil,
					AutoMerge:             false,
					TransientEnvironment:  false,
					ProductionEnvironment: strings.Contains(strings.ToLower(environment), "prod"),
				}
				deployment, _, err = scmClient.Deployments.Create(ctx, fullName, deploymentInput)
				if err != nil {
					return errors.Wrapf(err, "failed to create Deployment for repository %s and ref %s", fullName, ref)
				}
				log.Logger().Infof("created Deployment for release %s at %s", fullName, deployment.Link)
			}

			deploymentStatusInput := &scm.DeploymentStatusInput{
				State:           "success",
				TargetLink:      targetLink,
				LogLink:         logLink,
				Description:     description,
				Environment:     environment,
				EnvironmentLink: environmentLink,
				AutoInactive:    o.AutoInactive,
			}
			status, _, err := scmClient.Deployments.CreateStatus(ctx, fullName, deployment.ID, deploymentStatusInput)
			if err != nil {
				return errors.Wrapf(err, "failed to create DeploymentStatus for repository %s and ref %s", fullName, ref)
			}
			log.Logger().Infof("created DeploymentStatus for repository %s ref %s at %s with Logs URL %s and Target URL %s", fullName, ref, status.ID, logLink, targetLink)
		}
	}
	return nil
}

func (o *Options) CreateScmClient(group *v1alpha1.RepositoryGroup, repo *v1alpha1.Repository) (*scm.Client, error) {
	owner := group.Owner
	server := group.Provider
	if server == "" {
		return nil, errors.Errorf("no provider defined for owner %s", owner)
	}
	gitKind := group.ProviderKind
	if gitKind == "" {
		gitKind = giturl.SaasGitKind(server)
	}
	if gitKind == "" {
		return nil, errors.Errorf("no git provider kind for owner %s", owner)
	}

	// lets find the credentials from git...
	f := &scmhelpers.Factory{
		GitKind:      gitKind,
		GitServerURL: server,
		GitToken:     o.TestGitToken,
	}
	return f.Create()
}
