package tfupgrade

import (
	"io/ioutil"
	"net/url"
	"path/filepath"
	"regexp"
	"strings"

	jxc "github.com/jenkins-x/jx-api/v4/pkg/client/clientset/versioned"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/cli"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxclient"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube/jxenv"
	"github.com/jenkins-x/jx-helpers/v3/pkg/requirements"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/versionstream"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
)

var (
	info = termcolor.ColorInfo

	sourceRegex = regexp.MustCompile(`source\s*=\s*"(.*)"`)

	// gitURLAliases aliases for the version stream git URLs as terraform allows some different git URL layouts
	gitURLAliases = map[string]string{
		"jenkins-x/eks-jx/aws": "github.com/jenkins-x/terraform-aws-eks-jx",
	}
)

type Options struct {
	Dir              string
	VersionStreamDir string
	Namespace        string
	JXClient         jxc.Interface
	Resolver         *versionstream.VersionResolver
	GitClient        gitclient.Interface
	CommandRunner    cmdrunner.CommandRunner
}

// Validate validates the setup
func (o *Options) Validate() error {
	var err error
	if o.Dir == "" {
		o.Dir = "."
	}
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.QuietCommandRunner
	}
	if o.GitClient == nil {
		o.GitClient = cli.NewCLIClient("", o.CommandRunner)
	}
	o.JXClient, o.Namespace, err = jxclient.LazyCreateJXClientAndNamespace(o.JXClient, o.Namespace)
	if err != nil {
		return errors.Wrapf(err, "failed to create jx client")
	}
	return nil
}

func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate options")
	}
	path := filepath.Join(o.Dir, "main.tf")
	exists, err := files.FileExists(path)
	if err != nil {
		return errors.Wrapf(err, "failed to check for terraform file %s", path)
	}
	if !exists {
		return nil
	}

	log.Logger().Infof("checking for terraform git repository versions in file: %s", info(path))

	data, err := ioutil.ReadFile(path)
	if err != nil {
		return errors.Wrapf(err, "failed to load file %s", path)
	}

	// lets verify we have a resolver
	_, err = o.GetResolver()
	if err != nil {
		return errors.Wrapf(err, "failed to create a Resolver")
	}

	text := string(data)
	text2 := stringhelpers.ReplaceAllStringSubmatchFunc(sourceRegex, text, func(groups []stringhelpers.Group) []string {
		var answer []string
		for _, group := range groups {
			newValue := o.ReplaceValue(group.Value)
			if newValue == "" {
				newValue = group.Value
			}
			answer = append(answer, newValue)
		}
		return answer
	})

	if text2 == text {
		return nil

	}

	err = ioutil.WriteFile(path, []byte(text2), files.DefaultFileWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", path)
	}

	log.Logger().Infof("updated terraform git repository versions in: %s", info(path))
	return nil
}

func (o *Options) ReplaceValue(gitURL string) string {
	if strings.HasPrefix(gitURL, "git::") {
		answer := o.ReplaceValue(strings.TrimPrefix(gitURL, "git::"))
		if answer == "" {
			return ""
		}
		return "git::" + answer
	}
	u, err := url.Parse(gitURL)
	if err != nil {
		log.Logger().Infof("failed to parse terraform source URL %s due to: %s", gitURL, err.Error())
		return gitURL
	}
	ref := u.Query().Get("ref")

	plainGitURL := u.Path
	if u.Host != "" {
		plainGitURL = stringhelpers.UrlJoin(u.Host, u.Path)
	}

	version, err := o.findGitVersion(plainGitURL)
	if err != nil {
		log.Logger().Warnf("failed to resolve git version of URL %s due to: %s", plainGitURL, err.Error())
		return ""
	}
	if version == "" {
		return ""
	}
	if !strings.HasPrefix(version, "v") {
		version = "v" + version
	}
	if version == ref {
		return gitURL
	}
	query := u.Query()
	query.Set("ref", version)
	u.RawQuery = query.Encode()
	return u.String()
}

func (o *Options) findGitVersion(gitRepo string) (string, error) {
	resolver, err := o.GetResolver()
	if err != nil {
		return "", errors.Wrapf(err, "failed to create Resolver")
	}

	version, err := resolver.StableVersionNumber(versionstream.KindGit, gitRepo)
	if err != nil {
		return "", errors.Wrapf(err, "failed to resolve git version %s", gitRepo)
	}

	if version == "" {
		alias := gitURLAliases[gitRepo]
		if alias != "" {
			return o.findGitVersion(alias)
		}
	}
	return version, nil
}

func (o *Options) createResolver() (*versionstream.VersionResolver, error) {
	if o.VersionStreamDir == "" {
		path := filepath.Join(o.Dir, "versionStream")
		exists, err := files.DirExists(path)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to check for dir %s", path)
		}
		if exists {
			o.VersionStreamDir = path
		} else {
			// lets try find the dev environment git repository...
			gitURL := ""
			env, err := jxenv.GetDevEnvironment(o.JXClient, o.Namespace)
			if err != nil {
				return nil, errors.Wrap(err, "failed to get dev environment")
			}
			if env == nil {
				return nil, errors.Errorf("failed to find a dev environment source url as there is no 'dev' Environment resource in namespace %s", o.Namespace)
			}
			gitURL = env.Spec.Source.URL
			if gitURL == "" {
				return nil, errors.New("failed to find a dev environment source url on development environment resource")
			}
			_, dir, err := requirements.GetRequirementsAndGit(o.GitClient, gitURL)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to clone the cluster git repository %s", gitURL)
			}
			path = filepath.Join(dir, "versionStream")
			o.VersionStreamDir = path
			exists, err = files.DirExists(path)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to check for dir %s", path)
			}
			if !exists {
				return nil, errors.Errorf("the dev environment repository does not have a versionStream directory in clone %s", path)
			}
		}
	}

	if o.VersionStreamDir == "" {
		return nil, errors.Errorf("no version stream dir found")
	}
	return &versionstream.VersionResolver{
		VersionsDir: o.VersionStreamDir,
	}, nil
}

// GetResolver lazy creates the version stream resolver if we don't have one configured
func (o *Options) GetResolver() (*versionstream.VersionResolver, error) {
	if o.Resolver == nil {
		var err error
		o.Resolver, err = o.createResolver()
		if err != nil {
			return nil, errors.Wrapf(err, "failed to create the version resolver")
		}
	}
	return o.Resolver, nil
}
