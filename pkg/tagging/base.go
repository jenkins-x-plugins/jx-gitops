package tagging

import (
	"fmt"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	tagLong = templates.LongDesc(`
		%ss all kubernetes resources in the given directory tree
`)

	tagExample = templates.Examples(`
		# updates recursively %[1]ss all resources in the current directory 
		%s %[1]s my%[3]s=cheese another=thing
		# updates recursively all resources
		%[2]s %[1]s --dir myresource-dir foo=bar
		# remove %[3]ss
		%[2]s %[1]s my%[1]s- another-
	`)
)

// Options for the command
type Options struct {
	kyamls.Filter
	Dir       string
	PodSpec   bool
	Overwrite bool
}

// NewCmdUpdateTag creates a command object for the command
func NewCmdUpdateTag(tagVerb, tagType string) (*cobra.Command, *Options) {
	o := &Options{}
	caser := cases.Title(language.English)
	cmd := &cobra.Command{
		Use:     tagVerb,
		Short:   fmt.Sprintf("%ss all kubernetes resources in the given directory tree", caser.String(tagVerb)),
		Long:    fmt.Sprintf(tagLong, caser.String(tagVerb)),
		Example: fmt.Sprintf(tagExample, tagVerb, rootcmd.BinaryName, tagType),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.UpdateTagInYamlFiles(tagType+"s", args)
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "", ".", "the directory to recursively look for the *.yaml or *.yml files")
	cmd.Flags().BoolVarP(&o.PodSpec, "pod-spec", "p", false,
		fmt.Sprintf("%s the PodSpec in spec.template.metadata.%ss (or spec.jobTemplate.spec.template.metadata.%[2]ss for CronJobs) rather than the top level %[2]ss", tagVerb, tagType))
	cmd.Flags().BoolVar(&o.Overwrite, "overwrite", true, "Set to false to not overwrite any existing value")
	o.Filter.AddFlags(cmd)
	return cmd, o
}

// UpdateTagInYamlFiles updates the annotations in yaml files
func (o *Options) UpdateTagInYamlFiles(tagType string, tags []string) error {
	modifyFn := func(node *yaml.RNode, path string) (bool, error) {
		sort.Strings(tags)
		tagNode, err := getTagNode(node, path, tagType, o)
		if err != nil {
			return false, err
		}

		var modified bool
		for _, a := range tags {
			paths := strings.SplitN(a, "=", 2)
			key := paths[0]
			value := ""
			if len(paths) > 1 {
				value = paths[1]
			} else if strings.HasSuffix(key, "-") {
				modified, err = removeTag(tagNode, key, modified)
				if err != nil {
					return modified, err
				}
				continue
			}

			field, err := tagNode.Pipe(yaml.FieldMatcher{Name: key})
			if err != nil {
				return modified, errors.Wrapf(err, "failed to match %s", key)
			}
			valueNode := createValueNode(value)
			if field != nil {
				if !o.Overwrite {
					continue
				}
				field.SetYNode(valueNode)
				modified = true
			} else {
				addField(tagNode, key, valueNode)
				modified = true
			}
		}
		return modified, nil
	}

	return kyamls.ModifyFiles(o.Dir, modifyFn, o.Filter)
}

func getTagNode(node *yaml.RNode, path, tagType string, o *Options) (*yaml.RNode, error) {
	pathArray := []string{"metadata", tagType}
	if o.PodSpec {
		pathArray = append([]string{"spec", "template"}, pathArray...)
		if kyamls.GetKind(node, path) == "CronJob" {
			pathArray = append([]string{"spec", "jobTemplate"}, pathArray...)
		}
	}
	pathNode, err := node.Pipe(yaml.PathGetter{Path: pathArray, Create: yaml.MappingNode})
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get %v", pathArray)
	}
	return pathNode, nil
}

func addField(pathNode *yaml.RNode, key string, valueNode *yaml.Node) {
	pathNode.YNode().Content = append(
		pathNode.Content(),
		&yaml.Node{
			Kind:  yaml.ScalarNode,
			Value: key,
		},
		valueNode)
}

func createValueNode(value string) *yaml.Node {
	return &yaml.Node{
		Kind:  yaml.ScalarNode,
		Value: value,
		Tag:   yaml.NodeTagString,
		Style: yaml.SingleQuotedStyle,
	}
}

func removeTag(pathNode *yaml.RNode, k string, modified bool) (bool, error) {
	rNode, err := pathNode.Pipe(yaml.Clear(strings.TrimSuffix(k, "-")))
	if err != nil {
		return modified, err
	}
	modified = rNode != nil || modified
	return modified, nil
}
