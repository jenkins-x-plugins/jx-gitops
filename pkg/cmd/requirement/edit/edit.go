package edit

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/gitclient/giturl"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-helpers/v3/pkg/stringhelpers"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
)

// Options the CLI options for this command
type Options struct {
	Dir           string
	Requirements  jxcore.Requirements
	SecretStorage string
	Webhook       string
	Flags         RequirementBools
	Cmd           *cobra.Command
	Args          []string

	logsURL       string
	backupsURL    string
	reportsURL    string
	repositoryURL string
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
	o := &Options{
		Requirements: jxcore.Requirements{
			Spec: jxcore.RequirementsConfig{
				Ingress: jxcore.IngressConfig{
					TLS: &jxcore.TLSConfig{},
				},
			},
		},
	}
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
	cmd.Flags().BoolVarP(&o.Flags.VaultRecreateBucket, "vault-recreate-bucket", "", false, "enables or disables whether to rereate the secret bucket on boot")
	cmd.Flags().BoolVarP(&o.Flags.VaultDisableURLDiscover, "vault-disable-url-discover", "", false, "override the default lookup of the Vault URL, could be incluster service or external ingress")

	// requirements
	cmd.Flags().StringVarP(&o.SecretStorage, "secret", "s", "", fmt.Sprintf("configures the kind of secret storage. Values: %s", strings.Join(jxcore.SecretStorageTypeValues, ", ")))
	cmd.Flags().StringVarP(&o.Webhook, "webhook", "w", "", fmt.Sprintf("configures the kind of webhook. Values %s", strings.Join(jxcore.WebhookTypeValues, ", ")))

	// auto upgrade
	cmd.Flags().StringVarP(&o.Requirements.Spec.AutoUpdate.Schedule, "autoupdate-schedule", "", "", "the cron schedule for auto upgrading your cluster")

	// cluster
	cmd.Flags().StringVarP(&o.Requirements.Spec.Cluster.ClusterName, "cluster", "c", "", "configures the cluster name")
	cmd.Flags().StringVarP(&o.Requirements.Spec.Cluster.Provider, "provider", "p", "", "configures the kubernetes provider")
	cmd.Flags().StringVarP(&o.Requirements.Spec.Cluster.ProjectID, "project", "", "", "configures the Google Project ID")
	cmd.Flags().StringVarP(&o.Requirements.Spec.Cluster.Registry, "registry", "", "", "configures the host name of the container registry")
	cmd.Flags().StringVarP(&o.Requirements.Spec.Cluster.Region, "region", "r", "", "configures the cloud region")
	cmd.Flags().StringVarP(&o.Requirements.Spec.Cluster.Zone, "zone", "z", "", "configures the cloud zone")

	cmd.Flags().StringVarP(&o.Requirements.Spec.Cluster.ExternalDNSSAName, "extdns-sa", "", "", "configures the External DNS service account name")
	cmd.Flags().StringVarP(&o.Requirements.Spec.Cluster.KanikoSAName, "kaniko-sa", "", "", "configures the Kaniko service account name")

	// git
	cmd.Flags().StringVarP(&o.Requirements.Spec.Cluster.GitKind, "git-kind", "", "", fmt.Sprintf("the kind of git repository to use. Possible values: %s", strings.Join(giturl.KindGits, ", ")))
	cmd.Flags().StringVarP(&o.Requirements.Spec.Cluster.GitName, "git-name", "", "", "the name of the git repository")
	cmd.Flags().StringVarP(&o.Requirements.Spec.Cluster.GitServer, "git-server", "", "", "the git server host such as https://github.com or https://gitlab.com")
	cmd.Flags().StringVarP(&o.Requirements.Spec.Cluster.EnvironmentGitOwner, "env-git-owner", "", "", "the git owner (organisation or user) used to own the git repositories for the environments")

	// ingress
	cmd.Flags().StringVarP(&o.Requirements.Spec.Ingress.Domain, "domain", "d", "", "configures the domain name")
	cmd.Flags().StringVarP(&o.Requirements.Spec.Ingress.TLS.Email, "tls-email", "", "", "the TLS email address to enable TLS on the domain")

	// storage
	cmd.Flags().StringVarP(&o.logsURL, "bucket-logs", "", "", "the bucket URL to store logs")
	cmd.Flags().StringVarP(&o.backupsURL, "bucket-backups", "", "", "the bucket URL to store backups")
	cmd.Flags().StringVarP(&o.repositoryURL, "bucket-repo", "", "", "the bucket URL to store repository artifacts")
	cmd.Flags().StringVarP(&o.reportsURL, "bucket-reports", "", "", "the bucket URL to store reports. If not specified default to te logs bucket")

	// vault
	cmd.Flags().StringVarP(&o.Requirements.Spec.Vault.Name, "vault-name", "", "", "specify the vault name")
	cmd.Flags().StringVarP(&o.Requirements.Spec.Vault.Bucket, "vault-bucket", "", "", "specify the vault bucket")
	cmd.Flags().StringVarP(&o.Requirements.Spec.Vault.Keyring, "vault-keyring", "", "", "specify the vault key ring")
	cmd.Flags().StringVarP(&o.Requirements.Spec.Vault.Key, "vault-key", "", "", "specify the vault key")
	cmd.Flags().StringVarP(&o.Requirements.Spec.Vault.ServiceAccount, "vault-sa", "", "", "specify the vault Service Account name")

	return cmd, o
}

// Run runs the command
func (o *Options) Run() error {
	requirementsResource, fileName, err := jxcore.LoadRequirementsConfig(o.Dir, jxcore.DefaultFailOnValidationError)
	if err != nil {
		return err
	}

	if fileName == "" {
		fileName = filepath.Join(o.Dir, jxcore.RequirementsConfigFileName)
	}
	o.Requirements = *requirementsResource

	// lets re-parse the CLI arguments to re-populate the loaded requirements
	err = o.Cmd.Flags().Parse(os.Args)
	if err != nil {
		return errors.Wrap(err, "failed to reparse arguments")
	}

	config, err := o.applyDefaults()
	if err != nil {
		return err
	}

	requirementsResource.Spec = config
	err = requirementsResource.SaveConfig(fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save %s", fileName)
	}

	log.Logger().Infof("saved file: %s", termcolor.ColorInfo(fileName))
	return nil
}

func (o *Options) applyDefaults() (jxcore.RequirementsConfig, error) {
	r := o.Requirements.Spec

	gitKind := r.Cluster.GitKind
	if gitKind != "" && stringhelpers.StringArrayIndex(giturl.KindGits, gitKind) < 0 {
		return r, options.InvalidOption("git-kind", gitKind, giturl.KindGits)
	}

	// override boolean flags if specified
	if o.FlagChanged("autoupgrade") {
		r.AutoUpdate.Enabled = o.Flags.AutoUpgrade
	}
	if o.FlagChanged("env-git-public") {
		r.Cluster.EnvironmentGitPublic = o.Flags.EnvironmentGitPublic
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
			r.SecretStorage = jxcore.SecretStorageTypeLocal
		case "vault":
			r.SecretStorage = jxcore.SecretStorageTypeVault
		default:
			return r, options.InvalidOption("secret", o.SecretStorage, jxcore.SecretStorageTypeValues)
		}
	}
	if o.Webhook != "" {
		switch o.Webhook {
		case "lighthouse":
			r.Webhook = jxcore.WebhookTypeLighthouse
		default:
			return r, options.InvalidOption("webhook", o.Webhook, jxcore.WebhookTypeValues)
		}
	}

	// default flags if associated values
	if r.AutoUpdate.Schedule != "" {
		r.AutoUpdate.Enabled = true
	}
	if r.Ingress.TLS != nil && r.Ingress.TLS.Email != "" {
		r.Ingress.TLS.Enabled = true
	}

	// enable storage if we specify a URL
	if r.GetStorageURL("logs") != "" && r.GetStorageURL("reports") == "" {
		r.AddOrUpdateStorageURL("reports", r.GetStorageURL("logs"))
	}
	if o.logsURL != "" {
		r.AddOrUpdateStorageURL("logs", o.logsURL)
	}
	if o.backupsURL != "" {
		r.AddOrUpdateStorageURL("backup", o.backupsURL)
	}
	if o.reportsURL != "" {
		r.AddOrUpdateStorageURL("reports", o.reportsURL)
	}
	if o.repositoryURL != "" {
		r.AddOrUpdateStorageURL("repository", o.repositoryURL)
	}
	return r, nil
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
