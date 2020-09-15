package edit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/pkg/options"
	"github.com/jenkins-x/jx-helpers/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx-helpers/pkg/cobras/templates"
)

// Options the CLI options for this command
type Options struct {
	Dir           string
	Requirements  config.RequirementsConfig
	SecretStorage string
	Webhook       string
	Flags         RequirementBools
	Cmd           *cobra.Command
	Args          []string
}

// RequirementBools for the boolean flags we only update if specified on the CLI
type RequirementBools struct {
	AutoUpgrade, EnvironmentGitPublic, GitOps, Kaniko, Terraform bool
	VaultRecreateBucket, VaultDisableURLDiscover                 bool
}

var (
	cmdLong = templates.LongDesc(`
		Edits the local 'jx-requirements.yml file 
`)

	cmdExample = templates.Examples(`
		# edits the local 'jx-requirements.yml' file 
		%s requirements edit --domain foo.com --tls --provider eks
`)
)

// NewCmdRequirementsEdit creates the new command
func NewCmdRequirementsEdit() (*cobra.Command, *Options) {
	o := &Options{}
	cmd := &cobra.Command{
		Use:     "edit",
		Short:   "Edits the local 'jx-requirements.yml file",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			o.Cmd = cmd
			o.Args = args
			err := o.Run()
			helper.CheckErr(err)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			o.Cmd = cmd
			o.Args = args
			return o.Run()
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "", ".", "the directory to search for the 'jx-requirements.yml' file")

	// bools
	cmd.Flags().BoolVarP(&o.Flags.AutoUpgrade, "autoupgrade", "", false, "enables or disables auto upgrades")
	cmd.Flags().BoolVarP(&o.Flags.EnvironmentGitPublic, "env-git-public", "", false, "enables or disables whether the environment repositories should be public")
	cmd.Flags().BoolVarP(&o.Flags.GitOps, "gitops", "g", false, "enables or disables the use of gitops")
	cmd.Flags().BoolVarP(&o.Flags.Kaniko, "kaniko", "", false, "enables or disables the use of kaniko")
	cmd.Flags().BoolVarP(&o.Flags.Terraform, "terraform", "", false, "enables or disables the use of terraform")
	cmd.Flags().BoolVarP(&o.Flags.VaultRecreateBucket, "vault-recreate-bucket", "", false, "enables or disables whether to rereate the secret bucket on boot")
	cmd.Flags().BoolVarP(&o.Flags.VaultDisableURLDiscover, "vault-disable-url-discover", "", false, "override the default lookup of the Vault URL, could be incluster service or external ingress")

	// requirements
	cmd.Flags().StringVarP(&o.Requirements.BootConfigURL, "boot-config-url", "", "", "specify the boot configuration git URL")
	cmd.Flags().StringVarP(&o.SecretStorage, "secret", "s", "", fmt.Sprintf("configures the kind of secret storage. Values: %s", strings.Join(config.SecretStorageTypeValues, ", ")))
	cmd.Flags().StringVarP(&o.Webhook, "webhook", "w", "", fmt.Sprintf("configures the kind of webhook. Values %s", strings.Join(config.WebhookTypeValues, ", ")))

	// auto upgrade
	cmd.Flags().StringVarP(&o.Requirements.AutoUpdate.Schedule, "autoupdate-schedule", "", "", "the cron schedule for auto upgrading your cluster")

	// cluster
	cmd.Flags().StringVarP(&o.Requirements.Cluster.ClusterName, "cluster", "c", "", "configures the cluster name")
	cmd.Flags().StringVarP(&o.Requirements.Cluster.Namespace, "namespace", "n", "", "configures the namespace to use")
	cmd.Flags().StringVarP(&o.Requirements.Cluster.Provider, "provider", "p", "", "configures the kubernetes provider")
	cmd.Flags().StringVarP(&o.Requirements.Cluster.ProjectID, "project", "", "", "configures the Google Project ID")
	cmd.Flags().StringVarP(&o.Requirements.Cluster.Registry, "registry", "", "", "configures the host name of the container registry")
	cmd.Flags().StringVarP(&o.Requirements.Cluster.Region, "region", "r", "", "configures the cloud region")
	cmd.Flags().StringVarP(&o.Requirements.Cluster.Zone, "zone", "z", "", "configures the cloud zone")

	cmd.Flags().StringVarP(&o.Requirements.Cluster.ExternalDNSSAName, "extdns-sa", "", "", "configures the External DNS service account name")
	cmd.Flags().StringVarP(&o.Requirements.Cluster.KanikoSAName, "kaniko-sa", "", "", "configures the Kaniko service account name")
	cmd.Flags().StringVarP(&o.Requirements.Cluster.HelmMajorVersion, "helm-version", "", "", "configures the Helm major version. e.g. 3 to try helm 3")

	// git
	cmd.Flags().StringVarP(&o.Requirements.Cluster.GitKind, "git-kind", "", "", fmt.Sprintf("the kind of git repository to use. Possible values: %s", strings.Join(giturl.KindGits, ", ")))
	cmd.Flags().StringVarP(&o.Requirements.Cluster.GitName, "git-name", "", "", "the name of the git repository")
	cmd.Flags().StringVarP(&o.Requirements.Cluster.GitServer, "git-server", "", "", "the git server host such as https://github.com or https://gitlab.com")
	cmd.Flags().StringVarP(&o.Requirements.Cluster.EnvironmentGitOwner, "env-git-owner", "", "", "the git owner (organisation or user) used to own the git repositories for the environments")

	// ingress
	cmd.Flags().StringVarP(&o.Requirements.Ingress.Domain, "domain", "d", "", "configures the domain name")
	cmd.Flags().StringVarP(&o.Requirements.Ingress.TLS.Email, "tls-email", "", "", "the TLS email address to enable TLS on the domain")

	// storage
	cmd.Flags().StringVarP(&o.Requirements.Storage.Logs.URL, "bucket-logs", "", "", "the bucket URL to store logs")
	cmd.Flags().StringVarP(&o.Requirements.Storage.Backup.URL, "bucket-backups", "", "", "the bucket URL to store backups")
	cmd.Flags().StringVarP(&o.Requirements.Storage.Repository.URL, "bucket-repo", "", "", "the bucket URL to store repository artifacts")
	cmd.Flags().StringVarP(&o.Requirements.Storage.Reports.URL, "bucket-reports", "", "", "the bucket URL to store reports. If not specified default to te logs bucket")

	// vault
	cmd.Flags().StringVarP(&o.Requirements.Vault.Name, "vault-name", "", "", "specify the vault name")
	cmd.Flags().StringVarP(&o.Requirements.Vault.Bucket, "vault-bucket", "", "", "specify the vault bucket")
	cmd.Flags().StringVarP(&o.Requirements.Vault.Keyring, "vault-keyring", "", "", "specify the vault key ring")
	cmd.Flags().StringVarP(&o.Requirements.Vault.Key, "vault-key", "", "", "specify the vault key")
	cmd.Flags().StringVarP(&o.Requirements.Vault.ServiceAccount, "vault-sa", "", "", "specify the vault Service Account name")

	// velero
	cmd.Flags().StringVarP(&o.Requirements.Velero.ServiceAccount, "velero-sa", "", "", "specify the Velero Service Account name")
	cmd.Flags().StringVarP(&o.Requirements.Velero.Namespace, "velero-ns", "", "", "specify the Velero Namespace")

	// version stream
	cmd.Flags().StringVarP(&o.Requirements.VersionStream.URL, "version-stream-url", "", "", "specify the Version Stream git URL")
	cmd.Flags().StringVarP(&o.Requirements.VersionStream.Ref, "version-stream-ref", "", "", "specify the Version Stream git reference (branch, tag, sha)")
	return cmd, o
}

// Run runs the command
func (o *Options) Run() error {
	requirements, fileName, err := config.LoadRequirementsConfig(o.Dir, config.DefaultFailOnValidationError)
	if err != nil {
		return err
	}
	if fileName == "" {
		fileName = filepath.Join(o.Dir, config.RequirementsConfigFileName)
	}
	o.Requirements = *requirements

	// lets re-parse the CLI arguments to re-populate the loaded requirements
	err = o.Cmd.Flags().Parse(os.Args)
	if err != nil {
		return errors.Wrap(err, "failed to reparse arguments")
	}

	err = o.applyDefaults()
	if err != nil {
		return err
	}

	err = o.Requirements.SaveConfig(fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save %s", fileName)
	}

	log.Logger().Infof("saved file: %s", termcolor.ColorInfo(fileName))
	return nil
}

func (o *Options) applyDefaults() error {
	r := &o.Requirements

	gitKind := r.Cluster.GitKind
	if gitKind != "" && stringhelpers.StringArrayIndex(giturl.KindGits, gitKind) < 0 {
		return options.InvalidOption("git-kind", gitKind, giturl.KindGits)
	}

	// override boolean flags if specified
	if o.FlagChanged("autoupgrade") {
		r.AutoUpdate.Enabled = o.Flags.AutoUpgrade
	}
	if o.FlagChanged("env-git-public") {
		r.Cluster.EnvironmentGitPublic = o.Flags.EnvironmentGitPublic
	}
	if o.FlagChanged("gitops") {
		r.GitOps = o.Flags.GitOps
	}
	if o.FlagChanged("kaniko") {
		r.Kaniko = o.Flags.Kaniko
	}
	if o.FlagChanged("terraform") {
		r.Terraform = o.Flags.Terraform
	}
	if o.FlagChanged("vault-disable-url-discover") {
		r.Vault.DisableURLDiscovery = o.Flags.VaultDisableURLDiscover
	}
	if o.FlagChanged("vault-recreate-bucket") {
		r.Vault.RecreateBucket = o.Flags.VaultRecreateBucket
	}

	// custom string types...
	if o.SecretStorage != "" {
		switch o.SecretStorage {
		case "local":
			r.SecretStorage.Provider = config.SecretStorageTypeLocal
		case "vault":
			r.SecretStorage.Provider = config.SecretStorageTypeVault
		default:
			return options.InvalidOption("secret", o.SecretStorage, config.SecretStorageTypeValues)
		}
	}
	if o.Webhook != "" {
		switch o.Webhook {
		case "jenkins":
			r.Webhook = config.WebhookTypeJenkins
		case "lighthouse":
			r.Webhook = config.WebhookTypeLighthouse
		case "prow":
			r.Webhook = config.WebhookTypeProw
		default:
			return options.InvalidOption("webhook", o.Webhook, config.WebhookTypeValues)
		}
	}

	// default flags if associated values
	if r.AutoUpdate.Schedule != "" {
		r.AutoUpdate.Enabled = true
	}
	if r.Ingress.TLS.Email != "" {
		r.Ingress.TLS.Enabled = true
	}

	// enable storage if we specify a URL
	storage := &r.Storage
	if storage.Logs.URL != "" && storage.Reports.URL == "" {
		storage.Reports.URL = storage.Logs.URL
	}
	o.defaultStorage(&storage.Backup)
	o.defaultStorage(&storage.Logs)
	o.defaultStorage(&storage.Reports)
	o.defaultStorage(&storage.Repository)
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
