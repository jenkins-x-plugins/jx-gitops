package image

import (
	"fmt"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/versionstreamer"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	cmdLong = templates.LongDesc(`
		Updates images in the kubernetes resources from the version stream
`)

	cmdExample = templates.Examples(`
		# modify the images in the content-root folder using the current version stream
		%s image
		# modify the images in the ./src dir using the current dir to find the version stream
		%s image --source-dir ./src --dir . 
	`)

	kindToPaths = map[string][][]string{
		"Deployment": {
			{
				"spec", "template", "spec", "initContainers", "image",
			},
			{
				"spec", "template", "spec", "containers", "image",
			},
		},
		"Job": {
			{
				"spec", "template", "spec", "initContainers", "image",
			},
			{
				"spec", "template", "spec", "containers", "image",
			},
		},
		"Pipeline": {
			{
				"spec", "tasks", "taskSpec", "steps", "image",
			},
		},
		"PipelineRun": {
			{
				"spec", "pipelineSpec", "tasks", "taskSpec", "steps", "image",
			},
		},
		"Task": {
			{
				"spec", "steps", "image",
			},
		},
	}
)

// Options the options for the command
type Options struct {
	kyamls.Filter
	VersionStreamer versionstreamer.Options
	SourceDir       string
	ImageResolver   func(string, []string, string) (string, error)
	gitURL          string
	gitInfo         *giturl.GitRepository
}

// NewCmdUpdateImage creates a command object for the command
func NewCmdUpdateImage() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use: "image",

		Short:   "Updates images in the kubernetes resources from the version stream",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.SourceDir, "source-dir", "s", "content-root", "the directory to recursively look for the *.yaml files to modify")
	o.Filter.AddFlags(cmd)
	o.VersionStreamer.AddFlags(cmd)
	return cmd, o
}

// Run transforms the YAML files
func (o *Options) Run() error {
	err := o.VersionStreamer.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to create version stream resolver")
	}
	resolver := o.VersionStreamer.Resolver
	if resolver == nil {
		return errors.Errorf("no version stream resolver created")
	}

	if o.ImageResolver == nil {
		o.ImageResolver = o.resolveImage
	}
	modifyFn := func(node *yaml.RNode, path string) (bool, error) {
		kind := kyamls.GetKind(node, path)
		answer := false
		pathsSlice := kindToPaths[kind]
		if len(pathsSlice) > 0 {
			for _, jsonNames := range pathsSlice {
				flag, err := o.modifyImages(node, path, "", jsonNames...)
				if err != nil {
					return flag, err
				}
				if flag {
					answer = true
				}
			}
		}
		return answer, nil
	}
	return kyamls.ModifyFiles(o.SourceDir, modifyFn, o.Filter)
}

func (o *Options) modifyImages(node *yaml.RNode, filePath string, jsonPath string, names ...string) (bool, error) {
	if len(names) == 0 {
		return false, errors.Errorf("no JSON path names supplied")
	}
	flag := false
	fullJSONPath := kyamls.JSONPath(names...)
	if jsonPath != "" {
		fullJSONPath = kyamls.JSONPath(jsonPath, fullJSONPath)
	}

	if node.YNode().Kind == yaml.SequenceNode {
		err := node.VisitElements(func(sn *yaml.RNode) error {
			var err error
			flag, err = o.modifyImages(sn, filePath, jsonPath, names...)
			return err
		})
		if err != nil {
			return false, errors.Wrapf(err, "failed to process sequence at path %s for file %s", fullJSONPath, filePath)
		}
		return flag, nil
	}
	key := names[0]
	err := node.VisitFields(func(mn *yaml.MapNode) error {
		keyText, err := mn.Key.String()
		if err != nil {
			return errors.Wrapf(err, "failed to get key for path %s for file %s", jsonPath, filePath)
		}
		keyText = strings.TrimSpace(keyText)
		childJSONPath := keyText
		if jsonPath != "" {
			childJSONPath = kyamls.JSONPath(jsonPath, keyText)
		}
		if keyText != key {
			return nil
		}
		if len(names) == 1 {
			valueText, err := mn.Value.String()
			if err != nil {
				return errors.Wrapf(err, "failed to get the image value of %s for path %s for file %s", keyText, childJSONPath, filePath)
			}

			imageWithoutTag := strings.TrimSpace(valueText)
			idx := strings.LastIndex(imageWithoutTag, ":")
			if idx > 0 {
				imageWithoutTag = imageWithoutTag[0:idx]
			}
			newValue, err := o.ImageResolver(imageWithoutTag, names, filePath)
			if err != nil {
				return errors.Wrapf(err, "failed to get the image value of %s for path %s for file %s", keyText, childJSONPath, filePath)
			}
			if newValue != imageWithoutTag {
				mn.Value.SetYNode(&yaml.Node{Kind: yaml.ScalarNode, Value: newValue})
				log.Logger().Infof("modify %s: %s => %s for file %s", childJSONPath, valueText, newValue, filePath)
				flag = true
			} else {
				log.Logger().Debugf("not modifying %s: %s for file %s", childJSONPath, valueText, filePath)
			}
			return nil
		}

		flag, err = o.modifyImages(mn.Value, filePath, childJSONPath, names[1:]...)
		return err
	})
	if err != nil {
		return false, errors.Wrapf(err, "failed to navigate path %s for file %s", fullJSONPath, filePath)
	}
	return flag, nil
}

// resolveImage resolves the given container image from the version stream
func (o *Options) resolveImage(image string, names []string, filePath string) (string, error) {
	resolver := o.VersionStreamer.Resolver
	if resolver == nil {
		return "", errors.Errorf("cannot resolve image %s as no VersionResolver configured", image)
	}
	newImage, err := resolver.ResolveDockerImage(image)
	if err != nil {
		return "", errors.Wrapf(err, "failed to resolve image %s in the version stream at %s", image, resolver.VersionsDir)
	}
	return newImage, nil
}
