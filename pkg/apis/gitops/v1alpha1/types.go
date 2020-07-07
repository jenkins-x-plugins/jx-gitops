package v1alpha1

import (
	"gopkg.in/validator.v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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

	// DefaultBackendType the default back end to use if there's no specific mapping
	DefaultBackendType BackendType `json:"defaultBackendType,omitempty" validate:"nonzero"`
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
	// Project for the secret
	Project string `json:"project,omitempty"`
	// Mappings one more mappings
	Mappings []Mapping `json:"mappings,omitempty"`
}

// BackendType describes a secrets backend
type BackendType string

const (
	// BackendTypeVault Vault is the Backed service
	BackendTypeVault BackendType = "vault"
	// BackendTypeGSM Google Secrets Manager is the Backed service
	BackendTypeGSM BackendType = "gsm"
	// BackendTypeNone if none is configured
	BackendTypeNone BackendType = ""
)

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
		BackendType: c.Spec.DefaultBackendType,
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
