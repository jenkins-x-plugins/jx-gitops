package secret

import (
	"fmt"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	annotateLong = templates.LongDesc(`
		Adds one or more secrets to the given ServiceAccount files
`)

	annotateExample = templates.Examples(`
		# ensures that the given service account resource has the secret associated
		%s sa secret -f config-root/namespaces/jx/mychart/my-sa.yaml --secret my-secret-name
	`)
)

// AnnotateOptions the options for the command
type Options struct {
	kyamls.Filter
	File    string
	Secrets []string
}

// NewCmdServiceAccountSecrets creates a command object for the command
func NewCmdServiceAccountSecrets() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "secret",
		Short:   "Adds one or more secrets to the given ServiceAccount files",
		Long:    annotateLong,
		Example: fmt.Sprintf(annotateExample, rootcmd.BinaryName, rootcmd.BinaryName),
		Run: func(_ *cobra.Command, _ []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.File, "file", "f", "", "the ServiceAccount file to modify")
	cmd.Flags().StringArrayVarP(&o.Secrets, "secret", "s", nil, "the Secret names to add to the ServiceAccount")
	o.Filter.AddFlags(cmd)
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	path := o.File
	if path == "" {
		return options.MissingOption("file")
	}
	if len(o.Secrets) == 0 {
		return options.MissingOption("secret")
	}

	node, err := yaml.ReadFile(path)
	if err != nil {
		return errors.Wrapf(err, "failed to load file %s", path)
	}

	secretsPath := []string{"secrets"}
	secrets, err := node.Pipe(yaml.PathGetter{Path: secretsPath, Create: yaml.SequenceNode})
	if err != nil {
		return errors.Wrapf(err, "failed to create secrets field for file %s", path)
	}
	if secrets == nil {
		return errors.Errorf("failed to find secrets field for file %s", path)
	}
	elements, err := secrets.Elements()
	if err != nil {
		return errors.Wrapf(err, "failed to get elements for secrets node in %s", path)
	}

	var values []string
	for i, e := range elements {
		value, err := e.String()
		if err != nil {
			return errors.Wrapf(err, "failed to evaluate secret value %d in file %s", i, path)
		}
		values = append(values, value)
	}

	added := false
	content := secrets.Content()
	for _, secret := range o.Secrets {
		if stringhelpers.StringArrayIndex(values, secret) >= 0 {
			continue
		}

		// lets add a new element
		added = true
		secretNode := &yaml.Node{
			Kind: yaml.MappingNode,
		}
		err = yaml.NewRNode(secretNode).PipeE(yaml.FieldSetter{
			Name:        "name",
			StringValue: secret,
		})
		if err != nil {
			return errors.Wrapf(err, "failed to set name for secret %s on file %s", secret, path)
		}
		content = append(content, secretNode)
	}
	if !added {
		return nil
	}

	secrets.SetYNode(&yaml.Node{
		Kind:    yaml.SequenceNode,
		Content: content,
	})

	err = yamls.SaveFile(node, path)
	if err != nil {
		return errors.Wrapf(err, "failed to save %s", path)
	}
	info := termcolor.ColorInfo
	log.Logger().Infof("added secrets %s to ServiceAccount file %s", info(strings.Join(o.Secrets, " ")), info(path))
	return nil
}
