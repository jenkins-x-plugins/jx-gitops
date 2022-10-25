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
		%s %[1]s my%s[1]=cheese another=thing
		# updates recursively all resources
		%[2]s %[1]s --dir myresource-dir foo=bar
	`)
)

// Options for the command
type Options struct {
	kyamls.Filter
	Dir     string
	PodSpec bool
}

// NewCmdUpdateTag creates a command object for the command
func NewCmdUpdateTag(tagVerb, tagType string) (*cobra.Command, *Options) {
	o := &Options{}
	caser := cases.Title(language.English)
	cmd := &cobra.Command{
		Use:     tagVerb,
		Short:   fmt.Sprintf("%s all kubernetes resources in the given directory tree", caser.String(tagVerb)),
		Long:    fmt.Sprintf(tagLong, caser.String(tagVerb)),
		Example: fmt.Sprintf(tagExample, tagVerb, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := UpdateTagInYamlFiles(o.Dir, tagType, args, o.Filter, o.PodSpec)
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "", ".", "the directory to recursively look for the *.yaml or *.yml files")
	cmd.Flags().BoolVarP(&o.PodSpec, "pod-spec", "p", false,
		fmt.Sprintf("%s the PodSpec in spec.template.metadata.%s (or spec.jobTemplate.spec.template.metadata.%[2]s for CronJobs) rather than the top level %[2]s", tagVerb, tagType))
	o.Filter.AddFlags(cmd)
	return cmd, o
}

// UpdateTagInYamlFiles updates the annotations in yaml files
func UpdateTagInYamlFiles(dir string, tagType string, tags []string, filter kyamls.Filter, podSpec bool) error { //nolint:gocritic
	modifyFn := func(node *yaml.RNode, path string) (bool, error) {
		sort.Strings(tags)
		pathArray := []string{"metadata", tagType}
		if podSpec {
			pathArray = append([]string{"spec", "template"}, pathArray...)
			if kyamls.GetKind(node, path) == "CronJob" {
				pathArray = append([]string{"spec", "jobTemplate"}, pathArray...)
			}
		}

		for _, a := range tags {
			paths := strings.SplitN(a, "=", 2)
			k := paths[0]
			v := ""
			if len(paths) > 1 {
				v = paths[1]
			}

			vn := yaml.NewScalarRNode(v)
			vn.YNode().Tag = yaml.NodeTagString
			vn.YNode().Style = yaml.SingleQuotedStyle

			_, err := node.Pipe(
				yaml.PathGetter{Path: pathArray, Create: yaml.MappingNode},
				yaml.FieldSetter{Name: k, Value: vn})
			if err != nil {
				return false, errors.Wrapf(err, "failed to set %s=%s", k, v)
			}
		}
		return true, nil
	}

	return kyamls.ModifyFiles(dir, modifyFn, filter)
}
