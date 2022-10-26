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
			err := UpdateTagInYamlFiles(o.Dir, tagType+"s", args, o.Filter, o.PodSpec, o.Overwrite)
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
func UpdateTagInYamlFiles(dir string, tagType string, tags []string, filter kyamls.Filter, podSpec bool, override bool) error { //nolint:gocritic
	modifyFn := func(node *yaml.RNode, path string) (bool, error) {
		sort.Strings(tags)
		pathArray := []string{"metadata", tagType}
		if podSpec {
			pathArray = append([]string{"spec", "template"}, pathArray...)
			if kyamls.GetKind(node, path) == "CronJob" {
				pathArray = append([]string{"spec", "jobTemplate"}, pathArray...)
			}
		}
		pathNode, err := node.Pipe(yaml.PathGetter{Path: pathArray, Create: yaml.MappingNode})
		if err != nil {
			return false, errors.Wrapf(err, "failed to get %v", pathArray)
		}

		var modified bool
		for _, a := range tags {
			paths := strings.SplitN(a, "=", 2)
			k := paths[0]
			v := ""
			if len(paths) > 1 {
				v = paths[1]
			} else if strings.HasSuffix(k, "-") {
				rNode, err := pathNode.Pipe(yaml.Clear(strings.TrimSuffix(k, "-")))
				if err != nil {
					return modified, err
				}
				modified = rNode != nil
				continue
			}
			vn := yaml.NewScalarRNode(v)
			vn.YNode().Tag = yaml.NodeTagString
			vn.YNode().Style = yaml.SingleQuotedStyle

			field, err := pathNode.Pipe(yaml.FieldMatcher{Name: k})
			if err != nil {
				return modified, errors.Wrapf(err, "failed to match %s", k)
			}
			if field != nil {
				if !override {
					continue
				}
				// need to def ref the Node since field is ephemeral
				field.SetYNode(vn.YNode())
				modified = true
			} else {
				// create the field
				pathNode.YNode().Content = append(
					pathNode.Content(),
					&yaml.Node{
						Kind:  yaml.ScalarNode,
						Value: k,
					},
					vn.YNode())
				modified = true
			}
		}
		return modified, nil
	}

	return kyamls.ModifyFiles(dir, modifyFn, filter)
}
