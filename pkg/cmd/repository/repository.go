package repository

import (
	"fmt"

	"github.com/jenkins-x/jx-gitops/pkg/kyamls"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/jenkins-x/jx/v2/pkg/gits"
	"github.com/jenkins-x/jx/v2/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	labelLong = templates.LongDesc(`
		Updates the git repository URL for the cluster/environment
`)

	labelExample = templates.Examples(`
		# updates git repository URL for the resources in the current directory 
		%s repository https://github.com/myorg/myrepo.git
		# updates git repository URL for the resources in some directory 
		%s repository --dir something https://github.com/myorg/myrepo.git
	`)
)

// LabelOptions the options for the command
type Options struct {
	kyamls.Filter
	Dir     string
	Gitter  gits.Gitter
	gitURL  string
	gitInfo *gits.GitRepository
}

// NewCmdUpdateRepository creates a command object for the command
func NewCmdUpdateRepository() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "repository",
		Short:   "Updates the git repository URL for the cluster/environment",
		Long:    labelLong,
		Example: fmt.Sprintf(labelExample, rootcmd.BinaryName, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run(args)
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to recursively look for the *.yaml or *.yml files")
	o.Filter.AddFlags(cmd)
	return cmd, o
}

// Run transforms the YAML files
func (o *Options) Run(args []string) error {
	var err error
	discovered := false
	if len(args) > 0 {
		o.gitURL = args[0]
	}
	if len(args) == 0 {
		// lets try discover the git url
		o.gitURL, err = findGitURLFromDir(o.Git(), o.Dir)
		if err != nil {
			return errors.Wrapf(err, "failed to discover git URL in dir %s. you could try pass the git URL as an argument", o.Dir)
		}
		if o.gitURL == "" {
			return util.MissingArgument("git-url")
		}
		discovered = true
	}
	o.gitInfo, err = gits.ParseGitURL(o.gitURL)
	if err != nil {
		return errors.Wrapf(err, "failed to parse git URL: %s", o.gitURL)
	}
	if discovered {
		o.gitURL = o.gitInfo.URL

		log.Logger().Infof("discovered git URL %s replacing it in the dev Environment and Source Repository in dir %s", util.ColorInfo(o.gitURL), util.ColorInfo(o.Dir))
	}

	modifyFn := func(node *yaml.RNode, path string) (bool, error) {
		kind := kyamls.GetKind(node, path)
		answer := false
		switch kind {
		case "Environment":
			flag, err := o.modifyEnvironment(node, path)
			if err != nil {
				return flag, err
			}
			if flag {
				answer = true
			}
		case "SourceRepository":
			flag, err := o.modifySourceRepository(node, path)
			if err != nil {
				return flag, err
			}
			if flag {
				answer = true
			}
		}
		return answer, nil
	}
	return kyamls.ModifyFiles(o.Dir, modifyFn, o.Filter)
}

func (o *Options) modifyEnvironment(node *yaml.RNode, path string) (bool, error) {
	name := kyamls.GetName(node, path)
	if name != "dev" {
		return false, nil
	}
	err := node.PipeE(yaml.LookupCreate(yaml.ScalarNode, "spec", "source", "url"), yaml.FieldSetter{StringValue: o.gitURL})
	if err != nil {
		return false, errors.Wrapf(err, "failed to set the git source repository to %s for %s", o.gitURL, path)
	}
	return true, nil
}

func (o *Options) modifySourceRepository(node *yaml.RNode, path string) (bool, error) {
	name := kyamls.GetName(node, path)
	if name != "dev" {
		return false, nil
	}
	owner := o.gitInfo.Organisation
	err := node.PipeE(yaml.LookupCreate(yaml.ScalarNode, "spec", "org"), yaml.FieldSetter{StringValue: owner})
	if err != nil {
		return false, errors.Wrapf(err, "failed to set the git repository owner to %s for %s", owner, path)
	}
	repoName := o.gitInfo.Name
	err = node.PipeE(yaml.LookupCreate(yaml.ScalarNode, "spec", "repo"), yaml.FieldSetter{StringValue: repoName})
	if err != nil {
		return false, errors.Wrapf(err, "failed to set the git repository name to %s for %s", repoName, path)
	}
	err = node.PipeE(yaml.LookupCreate(yaml.ScalarNode, "spec", "httpCloneURL"), yaml.FieldSetter{StringValue: o.gitURL})
	if err != nil {
		return false, errors.Wrapf(err, "failed to set the git repository URL to %s for %s", o.gitURL, path)
	}
	return true, nil
}

// Git lazily create a gitter if its not specified
func (o *Options) Git() gits.Gitter {
	if o.Gitter == nil {
		o.Gitter = gits.NewGitCLI()
	}
	return o.Gitter
}

func findGitURLFromDir(gitter gits.Gitter, dir string) (string, error) {
	_, gitConfDir, err := gitter.FindGitConfigDir(dir)
	if err != nil {
		return "", errors.Wrapf(err, "there was a problem obtaining the git config dir of directory %s", dir)
	}

	if gitConfDir == "" {
		return "", fmt.Errorf("no .git directory could be found from dir %s", dir)
	}
	return gitter.DiscoverUpstreamGitURL(gitConfDir)
}
