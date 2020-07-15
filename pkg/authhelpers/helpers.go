package authhelpers

import (
	"os"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/jenkins-x/jx-helpers/pkg/kube"
	"github.com/jenkins-x/jx/v2/pkg/auth"
	"github.com/jenkins-x/jx/v2/pkg/gits"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
)

// AuthFacade a helper object for getting auth tokens
type AuthFacade struct {
	Service        auth.ConfigService
	Gitter         gits.Gitter
	IOFileHandles  *util.IOFileHandles
	BatchMode      bool
	UseGitHubOAuth bool
}

// NewAuthFacade creates a new auth facade
func NewAuthFacade() (*AuthFacade, error) {
	svc, err := LocalGitAuthService()
	if err != nil {
		return nil, err
	}
	return &AuthFacade{Service: svc}, nil
}

// NewAuthFacadeWithArgs creates a new auth facade with a set of arguments
func NewAuthFacadeWithArgs(svc auth.ConfigService, gitter gits.Gitter, ioFileHandles *util.IOFileHandles, batchMode bool, useGitHubOAuth bool) (*AuthFacade, error) {
	return &AuthFacade{
		Service:        svc,
		Gitter:         gitter,
		IOFileHandles:  ioFileHandles,
		BatchMode:      batchMode,
		UseGitHubOAuth: useGitHubOAuth,
	}, nil
}

func LocalGitAuthService() (auth.ConfigService, error) {
	return createAuthConfigServiceFile(auth.GitAuthConfigFile, kube.ValueKindGit)
}

func createAuthConfigServiceFile(fileName string, serverKind string) (auth.ConfigService, error) {
	authService, err := auth.NewFileAuthConfigService(fileName, serverKind)
	if err != nil {
		return nil, errors.Wrapf(err, "creating the auth config service from file %s", fileName)
	}
	if _, err := authService.LoadConfig(); err != nil {
		return nil, errors.Wrapf(err, "loading auth config from file %s", fileName)
	}
	return authService, nil
}

// Git lazily create a gitter if its not specified
func (f *AuthFacade) Git() gits.Gitter {
	if f.Gitter == nil {
		f.Gitter = gits.NewGitCLI()
	}
	return f.Gitter
}

// GetService lazily creates the auth service if required
func (f *AuthFacade) GetService() (auth.ConfigService, error) {
	if f.Service != nil {
		return f.Service, nil
	}
	var err error
	f.Service, err = LocalGitAuthService()
	return f.Service, err
}

// FindGitTokenForServer finds the git token and kind for the given server URL
func (f *AuthFacade) FindGitTokenForServer(serverURL string, owner string) (string, string, error) {
	_, token, kind, err := f.FindGitUserTokenForServer(serverURL, owner)
	return token, kind, err
}

// FindGitUserTokenForServer finds the git token and kind for the given server URL
func (f *AuthFacade) FindGitUserTokenForServer(serverURL string, owner string) (string, string, string, error) {
	user := ""
	token := ""
	kind := ""
	authSvc, err := f.GetService()
	if err != nil {
		return user, token, kind, errors.Wrapf(err, "failed to create the git auth service")
	}
	cfg := authSvc.Config()
	if cfg == nil {
		cfg, err = authSvc.LoadConfig()
		if err != nil {
			return user, token, kind, errors.Wrapf(err, "failed to load local git auth config")
		}
	}
	if cfg == nil {
		cfg = &auth.AuthConfig{}
	}
	server := cfg.GetOrCreateServer(serverURL)
	kind = server.Kind
	if kind == "" {
		kind = gits.SaasGitKind(serverURL)
	}

	ioHandles := GetIOFileHandles(f.IOFileHandles)

	userAuth, err := cfg.PickServerUserAuth(server, "Git user name:", f.BatchMode, owner, ioHandles)
	if err != nil {
		return user, token, kind, errors.Wrapf(err, "failed to pick git user name for server %s", serverURL)
	}

	if userAuth == nil || userAuth.IsInvalid() {
		fn := func(username string) error {
			f.Git().PrintCreateRepositoryGenerateAccessToken(server, username, ioHandles.Out)
			return nil
		}
		err = cfg.EditUserAuth(server.Label(), userAuth, userAuth.Username, false, f.BatchMode, fn, ioHandles)
		if err != nil {
			return user, token, kind, err
		}

		// TODO lets verify the auth works?
		if userAuth.IsInvalid() {
			return user, token, kind, errors.Wrapf(err, "authentication has failed for user %v. Please check the user's access credentials and try again", userAuth.Username)
		}

		err = authSvc.SaveUserAuth(server.URL, userAuth)
		if err != nil {
			return user, token, kind, errors.Wrapf(err, "failed to store git auth configuration")
		}
	}
	if userAuth == nil || userAuth.IsInvalid() {
		return user, token, kind, errors.Wrapf(err, "no valid token setup for git server %s", serverURL)
	}
	user = userAuth.Username
	token = userAuth.ApiToken
	if token == "" {
		token = userAuth.BearerToken
	}
	return user, token, kind, nil
}

// ScmClient creates a new Scm client for the given git server, owner and kind
func (f *AuthFacade) ScmClient(serverURL string, owner string, kind string) (*scm.Client, string, string, error) {
	login := ""
	token := ""
	if kind == "" || kind == "github" {
		kind = "github"
	}
	if login == "" || token == "" {
		u, t, defaultKind, err := f.FindGitUserTokenForServer(serverURL, owner)
		if err != nil {
			return nil, token, "", err
		}
		if login == "" {
			login = u
		}
		if token == "" {
			token = t
		}
		if kind == "" {
			kind = defaultKind
		}
	}

	client, err := factory.NewClient(kind, serverURL, token)
	return client, token, login, err
}

// GetIOFileHandles lazily creates a file handles object if the input is nil
func GetIOFileHandles(h *util.IOFileHandles) util.IOFileHandles {
	if h == nil {
		h = &util.IOFileHandles{
			Err: os.Stderr,
			In:  os.Stdin,
			Out: os.Stdout,
		}
	}
	return *h
}
