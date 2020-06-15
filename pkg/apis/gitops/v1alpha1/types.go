package v1alpha1

import (
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

	// Mappings one more mappings
	Mappings []Mapping `json:"mappings,omitempty"`
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
