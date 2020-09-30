package reqvalues

import (
	"github.com/jenkins-x/jx-api/pkg/config"
	"github.com/jenkins-x/jx-helpers/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/pkg/yamls"
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
)

const (
	// RequirementsValuesFileName is the name of the helm values.yaml configuration file for common Jenkins X values
	// such as cluster information, environments and ingress
	RequirementsValuesFileName = "jx-values.yaml"
)

// RequirementsValues contains the logical installation requirements in the `jx-requirements.yml` file as helm values
type RequirementsValues struct {
	// RequirementsConfig contains the logical installation requirements
	RequirementsConfig *config.RequirementsConfig `json:"jxRequirements,omitempty"`
}

// SaveRequirementsValuesFile saves the requirements yaml file for use with helmfile / helm 3
func SaveRequirementsValuesFile(c *config.RequirementsConfig, fileName string) error {
	y := &RequirementsValues{
		RequirementsConfig: c,
	}
	err := yamls.SaveFile(y, fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", fileName)
	}
	log.Logger().Debugf("generated helm YAML file from jx requirements at %s", termcolor.ColorInfo(fileName))
	return nil
}
