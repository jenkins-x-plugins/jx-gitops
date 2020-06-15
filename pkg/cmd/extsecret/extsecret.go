package extsecret

import (
	"fmt"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x/jx-gitops/pkg/common"
	"github.com/jenkins-x/jx-gitops/pkg/kyamls"
	"github.com/jenkins-x/jx-gitops/pkg/secretmapping"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

const (

	// BackendVault when using vault as the backend
	BackendVault = "vault"
)

var (
	labelLong = templates.LongDesc(`
		Converts all Secret resources in the path to ExternalSecret CRDs
`)

	labelExample = templates.Examples(`
		# updates recursively labels all resources in the current directory 
		%s extsecret --dir=.
	`)

	secretFilter = kyamls.Filter{
		Kinds: []string{"v1/Secret"},
	}
)

// LabelOptions the options for the command
type Options struct {
	Dir             string
	Backend         string
	VaultMountPoint string
	VaultRole       string
	SecretMapping   *v1alpha1.SecretMapping
}

// NewCmdExtSecrets creates a command object for the command
func NewCmdExtSecrets() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "extsecret",
		Aliases: []string{"extsecrets", "extsec"},
		Short:   "Converts all Secret resources in the path to ExternalSecret CRDs",
		Long:    labelLong,
		Example: fmt.Sprintf(labelExample, common.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to recursively look for the *.yaml or *.yml files")
	cmd.Flags().StringVarP(&o.Backend, "backend", "b", BackendVault, "the kind of external secret")
	cmd.Flags().StringVarP(&o.VaultMountPoint, "vault-mount-point", "m", "kubernetes", "the vault authentication mount point")
	cmd.Flags().StringVarP(&o.VaultRole, "vault-role", "r", "vault-infra", "the vault role that will be used to fetch the secrets. This role will need to be bound to kubernetes-external-secret's ServiceAccount; see Vault's documentation: https://www.vaultproject.io/docs/auth/kubernetes.html")
	return cmd, o
}

func (o *Options) Run() error {
	dir := o.Dir
	backend := o.Backend

	if o.SecretMapping == nil {
		var err error
		o.SecretMapping, _, err = secretmapping.LoadSecretMapping(dir, false)
		if err != nil {
			return errors.Wrapf(err, "failed to load secret mapping file")
		}
	}

	modifyFn := func(node *yaml.RNode, path string) (bool, error) {
		err := kyamls.SetStringValue(node, path, "kubernetes-client.io/v1", "apiVersion")
		if err != nil {
			return false, err
		}
		err = kyamls.SetStringValue(node, path, "ExternalSecret", "kind")
		if err != nil {
			return false, err
		}
		err = kyamls.SetStringValue(node, path, BackendVault, "spec", "backendType")
		if err != nil {
			return false, err
		}

		if backend == BackendVault {
			err = kyamls.SetStringValue(node, path, o.VaultMountPoint, "spec", "vaultMountPoint")
			if err != nil {
				return false, err
			}
			err = kyamls.SetStringValue(node, path, o.VaultRole, "spec", "vaultRole")
			if err != nil {
				return false, err
			}
		}

		flag, err := o.convertData(node, path)
		if err != nil {
			return flag, err
		}
		flag, err = o.moveMetadataToTemplate(node, path)
		if err != nil {
			return flag, err
		}
		return true, nil
	}
	return kyamls.ModifyFiles(dir, modifyFn, secretFilter)
}

func (o *Options) convertData(node *yaml.RNode, path string) (bool, error) {
	secretName := kyamls.GetStringField(node, path, "metadata", "name")

	data, err := node.Pipe(yaml.Lookup("data"))
	if err != nil {
		return false, errors.Wrapf(err, "failed to get data for path %s", path)
	}

	var contents []*yaml.Node
	style := node.Document().Style

	if data != nil {
		fields, err := data.Fields()
		if err != nil {
			return false, errors.Wrapf(err, "failed to find data fields for path %s", path)
		}
		for _, field := range fields {
			newNode := &yaml.Node{
				Kind:  yaml.MappingNode,
				Style: style,
			}

			rNode := yaml.NewRNode(newNode)

			// trim the suffix from the name and use it on the property?
			property := field
			secretPath := strings.ReplaceAll(secretName, "-", "/")
			key := "secret/data/" + secretPath

			if o.SecretMapping != nil {
				mapping := o.SecretMapping.Find(secretName, field)
				if mapping != nil {
					if mapping.Key != "" {
						key = mapping.Key
					}
					if mapping.Property != "" {
						property = mapping.Property
					}
				}
			}

			err = kyamls.SetStringValue(rNode, path, field, "name")
			if err != nil {
				return false, err
			}
			err = kyamls.SetStringValue(rNode, path, key, "key")
			if err != nil {
				return false, err
			}
			err = kyamls.SetStringValue(rNode, path, property, "property")
			if err != nil {
				return false, err
			}
			contents = append(contents, newNode)
		}
	}

	err = node.PipeE(yaml.Clear("data"))
	if err != nil {
		return false, errors.Wrapf(err, "failed to remove data")
	}
	data, err = node.Pipe(yaml.LookupCreate(yaml.SequenceNode, "spec", "data"))
	if err != nil {
		return false, errors.Wrapf(err, "failed to replace data for path %s", path)
	}
	if data == nil {
		return false, errors.Errorf("no data node for path %s", path)
	}
	data.SetYNode(&yaml.Node{
		Kind:    yaml.SequenceNode,
		Content: contents,
		Style:   style,
	})
	return true, nil
}

func (o *Options) moveMetadataToTemplate(node *yaml.RNode, path string) (bool, error) {
	// lets move annotations/labels/type  over to the template field
	typeValue := kyamls.GetStringField(node, path, "type")

	labels, err := node.Pipe(yaml.Lookup("metadata", "labels"))
	if err != nil {
		return false, errors.Wrapf(err, "failed to get labels")
	}
	annotations, err := node.Pipe(yaml.Lookup("metadata", "annotations"))
	if err != nil {
		return false, errors.Wrapf(err, "failed to get annotations")
	}

	if typeValue != "" || labels != nil || annotations != nil {
		templateNode, err := node.Pipe(yaml.LookupCreate(yaml.MappingNode, "spec", "template"))
		if err != nil {
			return false, errors.Wrapf(err, "failed to set kind")
		}
		if templateNode == nil {
			return false, errors.Errorf("could not create spec.template")
		}

		if annotations != nil {
			newAnnotations, err := templateNode.Pipe(yaml.LookupCreate(yaml.MappingNode, "metadata", "annotations"))
			if err != nil {
				return false, errors.Wrapf(err, "failed to set annotations on template")
			}
			newAnnotations.SetYNode(annotations.YNode())
		}
		if labels != nil {
			newLabels, err := templateNode.Pipe(yaml.LookupCreate(yaml.MappingNode, "metadata", "labels"))
			if err != nil {
				return false, errors.Wrapf(err, "failed to set annotations on template")
			}
			newLabels.SetYNode(labels.YNode())
		}
		if typeValue != "" {
			err = kyamls.SetStringValue(templateNode, path, typeValue, "type")
			if err != nil {
				return false, errors.Wrapf(err, "failed to set type on template")
			}
		}
		err = node.PipeE(yaml.Clear("type"))
		if err != nil {
			return false, errors.Wrapf(err, "failed to clear type")
		}
		metadata, err := node.Pipe(yaml.Lookup("metadata"))
		if err != nil {
			return false, errors.Wrapf(err, "failed to get metadata")
		}
		if metadata != nil {
			err = metadata.PipeE(yaml.Clear("annotations"))
			if err != nil {
				return false, errors.Wrapf(err, "failed to clear metadata annotations")
			}
			err = metadata.PipeE(yaml.Clear("labels"))
			if err != nil {
				return false, errors.Wrapf(err, "failed to clear metadata labels")
			}
		}
	}
	return true, nil
}
