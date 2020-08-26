package sourceconfigs

import (
	"sort"

	"github.com/jenkins-x/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x/jx-helpers/pkg/stringhelpers"
	"github.com/pkg/errors"
)

// DefaultValues defaults values from the given config, group and repository if they are missing
func DefaultValues(config *v1alpha1.SourceConfig, group *v1alpha1.RepositoryGroup, repo *v1alpha1.Repository) error {
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
	return nil
}

// GetOrCreateGroup get or create the group for the given name
func GetOrCreateGroup(config *v1alpha1.SourceConfig, owner string) *v1alpha1.RepositoryGroup {
	for i := range config.Spec.Groups {
		group := &config.Spec.Groups[i]
		if group.Owner == owner {
			return group
		}
	}
	config.Spec.Groups = append(config.Spec.Groups, v1alpha1.RepositoryGroup{
		Owner: owner,
	})
	return &config.Spec.Groups[len(config.Spec.Groups)-1]
}

// GetOrCreateRepository get or create the repository for the given name
func GetOrCreateRepository(config *v1alpha1.RepositoryGroup, repoName string) *v1alpha1.Repository {
	for i := range config.Repositories {
		repo := &config.Repositories[i]
		if repo.Name == repoName {
			return repo
		}
	}
	config.Repositories = append(config.Repositories, v1alpha1.Repository{
		Name: repoName,
	})
	return &config.Repositories[len(config.Repositories)-1]
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
