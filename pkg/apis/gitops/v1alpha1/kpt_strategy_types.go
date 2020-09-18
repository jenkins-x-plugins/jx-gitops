package v1alpha1

import (
	"gopkg.in/validator.v2"
)

const (
	// KptStragegyFileName default name of the kpt strategy file
	KptStragegyFileName = "kpt-strategy.yaml"
)

// +genclient
// +genclient:noStatus
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// KptStrategies contains a collection of merge strategies Jenkins X will use when performing kpt updates
//
// +k8s:openapi-gen=true
type KptStrategies struct {
	// KptStrategyConfig contains a collection of merge strategies Jenkins X will use when performing kpt updates
	KptStrategyConfig []KptStrategyConfig `json:"config" validate:"nonzero"`
}

// KptStrategyConfig used by jx gitops upgrade kpt
type KptStrategyConfig struct {
	// RelativePath the relative path to the folder the strategy should apply to
	RelativePath string `json:"relativePath" validate:"nonzero"`
	// Strategy is the merge strategy kpt will use see https://googlecontainertools.github.io/kpt/reference/pkg/update/#flags
	Strategy string `json:"strategy" validate:"nonzero"`
}

// validate the secrete mapping fields
func (c *KptStrategies) Validate() error {
	return validator.Validate(c)
}
