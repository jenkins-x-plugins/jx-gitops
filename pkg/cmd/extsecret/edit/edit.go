package edit

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/jenkins-x/jx-gitops/pkg/apis/gitops/v1alpha1"
	"github.com/jenkins-x/jx-gitops/pkg/secretmapping"

	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"

	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
)

// Options the CLI options for this command
type Options struct {
	Dir           string
	SecretMapping v1alpha1.SecretMapping
	Flags         SecretMappingOverrides
	Cmd           *cobra.Command
	Args          []string
}

// RequirementBools for the boolean flags we only update if specified on the CLI
type SecretMappingOverrides struct {
	ClusterName  string
	GCPProjectID string
}

const (
	flagGCPProjectID = "gcp-project-id"
	flagClusterName  = "cluster-name"
)

var (
	cmdLong = templates.LongDesc(`
		Edits the local 'secret-mappings.yaml' file 
`)

	cmdExample = templates.Examples(`
		# edits the local 'secret-mappings.yaml' file 
		%s extsecrets edit --gcp-project-id foo --cluster-name
`)
)

// NewCmdRequirementsEdit creates the new command
func NewCmdSecretMappingEdit() (*cobra.Command, *Options) {
	options := &Options{}
	cmd := &cobra.Command{
		Use:     "edit",
		Short:   "Edits the local 'secret-mappings.yaml' file",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			options.Cmd = cmd
			options.Args = args
			err := options.Run()
			helper.CheckErr(err)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			options.Cmd = cmd
			options.Args = args
			return options.Run()
		},
	}
	cmd.Flags().StringVarP(&options.Dir, "dir", "", "", "override the default '.jx/gitops/' dir to find the 'secret-mappings.yaml' file")
	cmd.Flags().StringVarP(&options.Flags.ClusterName, flagClusterName, "", "", "the cluster name")
	cmd.Flags().StringVarP(&options.Flags.GCPProjectID, flagGCPProjectID, "", "", "if GCP this is the project ID that hosts GSM secrets")

	return cmd, options
}

// Run runs the command
func (o *Options) Run() error {

	if o.Dir == "" {
		o.Dir = filepath.Join(".jx", "gitops")
	}

	secretMapping, fileName, err := secretmapping.LoadSecretMapping(o.Dir, true)
	if err != nil {
		return err
	}
	if fileName == "" {
		fileName = filepath.Join(o.Dir, v1alpha1.SecretMappingFileName)
	}
	o.SecretMapping = *secretMapping

	// lets re-parse the CLI arguments to re-populate the loaded requirements
	err = o.Cmd.Flags().Parse(os.Args)
	if err != nil {
		return errors.Wrap(err, "failed to reparse arguments")
	}

	err = o.applyDefaults()
	if err != nil {
		return err
	}

	err = o.SecretMapping.SaveConfig(fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save %s", fileName)
	}

	log.Logger().Infof("saved file: %s", termcolor.ColorInfo(fileName))
	return nil
}

func (o *Options) applyDefaults() error {
	s := &o.SecretMapping
	for k, secret := range s.Spec.Secrets {
		if secret.GcpSecretsManager == nil {
			secret.GcpSecretsManager = &v1alpha1.GcpSecretsManager{}
		}
		if secret.GcpSecretsManager.ProjectId == "" {
			if o.Flags.GCPProjectID == "" {
				return fmt.Errorf("found an empty gcp project id and no %s flag", flagGCPProjectID)
			}
			secret.GcpSecretsManager.ProjectId = o.Flags.GCPProjectID
		}
		if secret.GcpSecretsManager.UniquePrefix == "" {
			if o.Flags.ClusterName == "" {
				return fmt.Errorf("found an empty gcp unique prefix and no %s flag", flagClusterName)
			}
			secret.GcpSecretsManager.UniquePrefix = o.Flags.ClusterName
		}
		if secret.GcpSecretsManager.Version == "" {
			secret.GcpSecretsManager.Version = "latest"
		}
		s.Spec.Secrets[k] = secret
	}
	return nil
}

// FlagChanged returns true if the given flag was supplied on the command line
func (o *Options) FlagChanged(name string) bool {
	if o.Cmd != nil {
		f := o.Cmd.Flag(name)
		if f != nil {
			return f.Changed
		}
	}
	return false
}

func (o *Options) defaultStorage(storage *config.StorageEntryConfig) {
	if storage.URL != "" {
		storage.Enabled = true
	}
}
