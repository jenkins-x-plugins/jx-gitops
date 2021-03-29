package sourceconfigs

import (
	"os"
	"path/filepath"
	"sort"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
)

var info = termcolor.ColorInfo

// LoadSourceConfig loads the source config and optionally adds the default vlaues
func LoadSourceConfig(dir string, applyDefaults bool) (*v1alpha1.SourceConfig, error) {
	config := &v1alpha1.SourceConfig{}
	path := filepath.Join(dir, ".jx", "gitops", v1alpha1.SourceConfigFileName)

	exists, err := files.FileExists(path)
	if err != nil {
		return config, errors.Wrapf(err, "failed to check if file exists %s", path)
	}
	if !exists {
		log.Logger().Infof("the source config file %s does not exist", info(path))
		return config, nil
	}

	err = yamls.LoadFile(path, config)
	if err != nil {
		return config, errors.Wrapf(err, "failed to load file %s", path)
	}

	if applyDefaults {
		DefaultConfigValues(config)
	}
	return config, nil
}

// SaveSourceConfig saves the source config to the given directory
func SaveSourceConfig(config *v1alpha1.SourceConfig, dir string) error {
	if config.APIVersion == "" {
		config.APIVersion = v1alpha1.APIVersion
	}
	if config.Kind == "" {
		config.Kind = v1alpha1.KindSourceConfig
	}
	outDir := filepath.Join(dir, ".jx", "gitops")
	err := os.MkdirAll(outDir, files.DefaultDirWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to make directory %s", outDir)
	}

	path := filepath.Join(outDir, v1alpha1.SourceConfigFileName)
	err = yamls.SaveFile(config, path)
	if err != nil {
		return errors.Wrapf(err, "failed to save %s", path)
	}
	return nil
}

// DefaultConfigValues defaults values from the given config, group and repository if they are missing
func DefaultConfigValues(config *v1alpha1.SourceConfig) error {
	DefaultGroupValues(config, config.Spec.Groups)
	for i := range config.Spec.JenkinsServers {
		jenkinsServer := &config.Spec.JenkinsServers[i]
		DefaultGroupValues(config, jenkinsServer.Groups)
	}
	return nil
}

// DefaultGroupValues defaults values from the given config, group and repository if they are missing
func DefaultGroupValues(config *v1alpha1.SourceConfig, groups []v1alpha1.RepositoryGroup) error {
	for i := range groups {
		group := &groups[i]
		for j := range group.Repositories {
			repo := &group.Repositories[j]
			DefaultValues(config, group, repo)
		}
	}
	return nil
}

// DefaultValues defaults values from the given config, group and repository if they are missing
func DefaultValues(config *v1alpha1.SourceConfig, group *v1alpha1.RepositoryGroup, repo *v1alpha1.Repository) error {
	group.Slack = group.Slack.Inherit(config.Spec.Slack)

	if group.Provider == "" {
		group.Provider = "https://github.com"
	}
	if group.ProviderKind == "" {
		group.ProviderKind = "github"
	}
	if group.ProviderName == "" {
		group.ProviderName = "github"
	}
	if group.Scheduler == "" {
		group.Scheduler = config.Spec.Scheduler
	}

	if group.Owner == "" {
		return errors.Errorf("missing group.owner")
	}
	if repo.Name == "" {
		return errors.Errorf("missing repo.name")
	}
	if repo.URL == "" {
		repo.URL = stringhelpers.UrlJoin(group.Provider, group.Owner, repo.Name)
	}
	if repo.HTTPCloneURL == "" {
		repo.HTTPCloneURL = stringhelpers.UrlJoin(group.Provider, group.Owner, repo.Name+".git")
	}
	if repo.Scheduler == "" {
		repo.Scheduler = group.Scheduler
	}
	repo.Slack = repo.Slack.Inherit(group.Slack)
	return nil
}

// GetOrCreateGroup get or create the group for the given name
func GetOrCreateGroup(config *v1alpha1.SourceConfig, gitKind string, gitServerURL string, owner string) *v1alpha1.RepositoryGroup {
	var group *v1alpha1.RepositoryGroup
	config.Spec.Groups, group = getOrCreateGroup(config.Spec.Groups, gitKind, gitServerURL, owner)
	return group
}

// GetOrCreateJenkinsServerGroup get or create the group for the given name
func GetOrCreateJenkinsServerGroup(config *v1alpha1.JenkinsServer, gitKind string, gitServerURL string, owner string) *v1alpha1.RepositoryGroup {
	var group *v1alpha1.RepositoryGroup
	config.Groups, group = getOrCreateGroup(config.Groups, gitKind, gitServerURL, owner)
	return group
}

// getOrCreateGroup get or create the group for the given name
func getOrCreateGroup(groups []v1alpha1.RepositoryGroup, gitKind string, gitServerURL string, owner string) ([]v1alpha1.RepositoryGroup, *v1alpha1.RepositoryGroup) {
	for i := range groups {
		group := &groups[i]
		if (group.ProviderKind == gitKind || gitKind == "") && (group.Provider == gitServerURL || gitServerURL == "") && group.Owner == owner {
			return groups, group
		}
	}
	groups = append(groups, v1alpha1.RepositoryGroup{
		ProviderKind: gitKind,
		Provider:     gitServerURL,
		Owner:        owner,
	})
	return groups, &groups[len(groups)-1]
}

// GetOrCreateRepositoryFor returns the repository for the given git server URL if specified, owner and repository
func GetOrCreateRepositoryFor(config *v1alpha1.SourceConfig, gitServerURL, owner, repo string) *v1alpha1.Repository {
	group := GetOrCreateGroup(config, "", gitServerURL, owner)
	return GetOrCreateRepository(group, repo)
}

// GetOrCreateRepository get or create the repository for the given name
func GetOrCreateRepository(group *v1alpha1.RepositoryGroup, repoName string) *v1alpha1.Repository {
	for i := range group.Repositories {
		repo := &group.Repositories[i]
		if repo.Name == repoName {
			return repo
		}
	}
	group.Repositories = append(group.Repositories, v1alpha1.Repository{
		Name: repoName,
	})
	return &group.Repositories[len(group.Repositories)-1]
}

// GetOrCreateJenkinsServer get or create the jenkins server with the the given name
func GetOrCreateJenkinsServer(config *v1alpha1.SourceConfig, name string) *v1alpha1.JenkinsServer {
	for i := range config.Spec.JenkinsServers {
		js := &config.Spec.JenkinsServers[i]
		if js.Server == name {
			return js
		}
	}
	config.Spec.JenkinsServers = append(config.Spec.JenkinsServers, v1alpha1.JenkinsServer{
		Server: name,
	})
	return &config.Spec.JenkinsServers[len(config.Spec.JenkinsServers)-1]
}

// SortConfig sorts the repositories in each group
func SortConfig(config *v1alpha1.SourceConfig) {
	for i := range config.Spec.Groups {
		group := &config.Spec.Groups[i]
		SortRepositories(group.Repositories)
	}
}

// SortRepositories sorts the repositories
func SortRepositories(repositories []v1alpha1.Repository) {
	sort.Slice(repositories, func(i, j int) bool {
		r1 := repositories[i]
		r2 := repositories[j]
		return r1.Name < r2.Name
	})
}

// EnrichConfig ensures everything is populated
func EnrichConfig(config *v1alpha1.SourceConfig) {
	if config.APIVersion == "" {
		config.APIVersion = v1alpha1.APIVersion
	}
	if config.Kind == "" {
		config.Kind = v1alpha1.KindSourceConfig
	}

	// lets add a default slack configuration if it doesn't exist
	if config.Spec.Slack == nil {
		config.Spec.Slack = DefaultSlackNotify()
	}
}

func DefaultSlackNotify() *v1alpha1.SlackNotify {
	return &v1alpha1.SlackNotify{
		Channel:  v1alpha1.DefaultSlackChannel,
		Kind:     v1alpha1.NotifyKindFailureOrFirstSuccess,
		Pipeline: v1alpha1.PipelineKindRelease,
	}
}

func DryConfig(config *v1alpha1.SourceConfig) {
	// if all of the repositories in a group have the same scheduler then clear them all and set it on the group
	for i := range config.Spec.Groups {
		group := &config.Spec.Groups[i]
		scheduler := ""
		for j := range group.Repositories {
			repo := &group.Repositories[j]
			if repo.Scheduler == "" {
				scheduler = ""
				break
			}
			if scheduler == "" {
				scheduler = repo.Scheduler
			} else if scheduler != repo.Scheduler {
				scheduler = ""
				break
			}
		}
		if scheduler != "" {
			group.Scheduler = scheduler
			for j := range group.Repositories {
				group.Repositories[j].Scheduler = ""
			}
		}
	}

	// if the URLs can be guessed from the group, omit them
	for i := range config.Spec.Groups {
		group := &config.Spec.Groups[i]
		provider := group.Provider
		if provider == "" {
			break
		}
		owner := group.Owner
		for j := range group.Repositories {
			repo := &group.Repositories[j]
			name := repo.Name
			url := stringhelpers.UrlJoin(provider, owner, name)
			cloneURL := url + ".git"
			description := "Imported application for " + owner + "/" + name
			if repo.URL == url || repo.URL == cloneURL {
				repo.URL = ""
			}
			if repo.HTTPCloneURL == cloneURL {
				repo.HTTPCloneURL = ""
			}
			if repo.Description == description {
				repo.Description = ""
			}
		}
	}
}

// FindSettings finds the settings for the given owner and repository name
func FindSettings(config *v1alpha1.SourceConfig, owner string, repoName string) *jxcore.SettingsConfig {
	if owner == "" {
		owner = os.Getenv("REPO_OWNER")
	}
	if repoName == "" {
		repoName = os.Getenv("REPO_NAME")
	}

	// lets try find the group for the repository name
	for i := range config.Spec.Groups {
		group := &config.Spec.Groups[i]
		if group.Owner != owner {
			continue
		}
		for j := range group.Repositories {
			repo := &group.Repositories[j]
			if repo.Name == repoName {
				return group.Settings
			}
		}
	}

	// if the repo name can't be found then lets just find the first group for this owner
	for i := range config.Spec.Groups {
		group := &config.Spec.Groups[i]
		if group.Owner == owner {
			return group.Settings
		}
	}
	return nil
}
