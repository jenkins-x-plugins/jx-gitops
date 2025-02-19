package resolve

import (
	"fmt"
	"path/filepath"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-api/v4/pkg/util"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/gitdiscovery"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	labelLong = templates.LongDesc(`
		Resolves the git repository URL for the cluster/environment
`)

	labelExample = templates.Examples(`
		# updates git repository URL for the resources in the current directory 
		%s repository resolve https://github.com/myorg/myrepo.git
		# updates git repository URL for the resources in some directory 
		%[1]s repository resolve --dir something https://github.com/myorg/myrepo.git
	`)
)

// LabelOptions the options for the command
type Options struct {
	kyamls.Filter
	Dir       string
	SourceDir string
	gitURL    string
	gitInfo   *giturl.GitRepository
}

// NewCmdResolveRepository creates a command object for the command
func NewCmdResolveRepository() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "resolve",
		Short:   "Resolves the git repository URL for the cluster/environment",
		Long:    labelLong,
		Example: fmt.Sprintf(labelExample, rootcmd.BinaryName),
		Run: func(_ *cobra.Command, args []string) {
			err := o.Run(args)
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory look for the 'jx-requirements.yml` file")
	cmd.Flags().StringVarP(&o.SourceDir, "source-dir", "s", ".", "the directory to recursively look for the *.yaml or *.yml source Environment/SourceRepository files")
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
		o.gitURL, err = gitdiscovery.FindGitURLFromDir(o.SourceDir, false)
		if err != nil {
			return errors.Wrapf(err, "failed to discover git URL in dir %s. you could try pass the git URL as an argument", o.SourceDir)
		}
		if o.gitURL == "" {
			return options.MissingOption("git-url")
		}
		discovered = true
	}
	o.gitInfo, err = giturl.ParseGitURL(o.gitURL)
	if err != nil {
		return errors.Wrapf(err, "failed to parse git URL: %s", o.gitURL)
	}
	err = o.modifyRequirements()
	if err != nil {
		return errors.Wrapf(err, "failed to modify 'jx-requirements.yml'")
	}

	if discovered {
		o.gitURL = o.gitInfo.URL

		log.Logger().Debugf("discovered git URL %s replacing it in the dev Environment and Source Repository in dir %s", termcolor.ColorInfo(stringhelpers.SanitizeURL(o.gitURL)), termcolor.ColorInfo(o.SourceDir))
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
	return kyamls.ModifyFiles(o.SourceDir, modifyFn, o.Filter)
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

func (o *Options) modifyRequirements() error {
	dir := o.Dir
	fileName := filepath.Join(dir, jxcore.RequirementsConfigFileName)
	exists, err := util.FileExists(fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to check if file exists %s", fileName)
	}
	if !exists {
		log.Logger().Infof("no jx requirements file at %s", fileName)
		return nil
	}

	requirementsResource, err := jxcore.LoadRequirementsConfigFile(fileName, true)
	if err != nil {
		return errors.Wrapf(err, "failed to load file %s", fileName)
	}
	requirements := &requirementsResource.Spec

	repository := o.gitInfo.Name
	owner := o.gitInfo.Organisation
	log.Logger().Debugf("modifying jx-requirements.yml in dir %s to set the dev environment git repository to be %s/%s", dir, owner, repository)

	modified := false
	for i := range requirements.Environments {
		env := requirements.Environments[i]
		if env.Key == "dev" {
			requirements.Environments[i].Repository = repository
			requirements.Environments[i].Owner = owner
			modified = true
		}
	}
	if !modified {
		return errors.Errorf("could not find a 'dev' environment in the file %s", fileName)
	}
	log.Logger().Debugf("saving %s", fileName)
	err = requirementsResource.SaveConfig(fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", fileName)
	}
	return nil
}
