package v1alpha1

import (
	"io/ioutil"

	"github.com/jenkins-x/jx-api/pkg/util"

	"github.com/pkg/errors"
	"gopkg.in/validator.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"
)

const (
	SecretMappingFileName = "secret-mappings.yaml"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// SecretMapping represents a collection of mappings of Secrets to destinations in the underlying secret store (e.g. Vault keys)
//
// +k8s:openapi-gen=true
type SecretMapping struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ObjectMeta `json:"metadata"`

	// Spec holds the desired state of the SecretMapping from the client
	// +optional
	Spec SecretMappingSpec `json:"spec"`
}

// SecretMappingSpec defines the desired state of SecretMapping.
type SecretMappingSpec struct {
	// Secrets rules for each secret
	Secrets []SecretRule `json:"secrets,omitempty"`

	Defaults `json:"defaults,omitempty" validate:"nonzero"`
}

// Defaults contains default mapping configuration for any Kubernetes secrets to External Secrets
type Defaults struct {
	// DefaultBackendType the default back end to use if there's no specific mapping
	BackendType BackendType `json:"backendType,omitempty" validate:"nonzero"`
	// GcpSecretsManager config
	GcpSecretsManager GcpSecretsManager `json:"gcpSecretsManager,omitempty"`
}

// SecretMappingList contains a list of SecretMapping
//
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
type SecretMappingList struct {
	metav1.TypeMeta `json:",inline"`
	// +optional
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []SecretMapping `json:"items"`
}

// SecretRule the rules for a specific Secret
type SecretRule struct {
	// Name name of the secret
	Name string `json:"name,omitempty"`
	// Namespace name of the secret
	Namespace string `json:"namespace,omitempty"`
	// BackendType for the secret
	BackendType BackendType `json:"backendType"`
	// Mappings one more mappings
	Mappings []Mapping `json:"mappings,omitempty"`
	// Mandatory marks this secret as being mandatory
	Mandatory bool `json:"mandatory,omitempty"`
	// GcpSecretsManager config
	GcpSecretsManager GcpSecretsManager `json:"gcpSecretsManager,omitempty"`
}

// BackendType describes a secrets backend
type BackendType string

const (
	// BackendTypeVault Vault is the Backed service
	BackendTypeVault BackendType = "vault"
	// BackendTypeGSM Google Secrets Manager is the Backed service
	BackendTypeGSM BackendType = "gcpSecretsManager"
	// BackendTypeNone if none is configured
	BackendTypeNone BackendType = ""
)

// GcpSecretsManager the predicates which must be true to invoke the associated tasks/pipelines
type GcpSecretsManager struct {
	// Version of the referenced secret
	Version string `json:"version,omitempty"`
	// ProjectId for the secret, defaults to the current GCP project
	ProjectId string `json:"projectId,omitempty"`
	// UniquePrefix needs to be a unique prefix in the GCP project where the secret resides, defaults to cluster name
	UniquePrefix string `json:"uniquePrefix,omitempty"`
}

// Mapping the predicates which must be true to invoke the associated tasks/pipelines
type Mapping struct {
	// Name the secret entry name which maps to the Key of the Secret.Data map
	Name string `json:"name,omitempty"`

	// Key the Vault key to load the secret value
	// +optional
	Key string `json:"key,omitempty"`

	// Property the Vault property on the key to load the secret value
	// +optional
	Property string `json:"property,omitempty"`
}

// FindRule finds a secret rule for the given secret name
func (c *SecretMapping) FindRule(namespace string, secretName string) SecretRule {
	for _, m := range c.Spec.Secrets {
		if m.Name == secretName && (m.Namespace == "" || m.Namespace == namespace) {
			return m
		}
	}
	return SecretRule{
		BackendType: c.Spec.Defaults.BackendType,
	}
}

// Find finds a secret rule for the given secret name
func (c *SecretMapping) Find(secretName string, dataKey string) *Mapping {
	for i, m := range c.Spec.Secrets {
		if m.Name == secretName {
			return c.Spec.Secrets[i].Find(dataKey)
		}
	}
	return nil
}

// Find finds a secret rule for the given secret name
func (c *SecretMapping) FindSecret(secretName string) *SecretRule {
	for i, m := range c.Spec.Secrets {
		if m.Name == secretName {
			return &c.Spec.Secrets[i]
		}
	}
	return nil
}

// Find finds a mapping for the given data name
func (r *SecretRule) Find(dataKey string) *Mapping {
	for i, m := range r.Mappings {
		if m.Name == dataKey {
			return &r.Mappings[i]
		}
	}
	return nil
}

// validate the secrete mapping fields
func (c *SecretMapping) Validate() error {
	return validator.Validate(c)
}

// SaveConfig saves the configuration file to the given project directory
func (c *SecretMapping) SaveConfig(fileName string) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(fileName, data, util.DefaultWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", fileName)
	}

	return nil
}
