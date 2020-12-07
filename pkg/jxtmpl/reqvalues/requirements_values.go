package reqvalues

import (
	"path/filepath"

	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/maps"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
)

const (
	// RequirementsValuesFileName is the name of the helm values.yaml configuration file for common Jenkins X values
	// such as cluster information, environments and ingress
	RequirementsValuesFileName = "jx-values.yaml"
)

type HelmfileConditional struct {
	Enabled bool `json:"enabled"`
}

// RequirementsValues contains the logical installation requirements in the `jx-requirements.yml` file as helm values
type RequirementsValues struct {
	// RequirementsConfig contains the logical installation requirements
	RequirementsConfig          *jxcore.RequirementsConfig `json:"jxRequirements,omitempty"`
	IngressExternalDNSCondition *HelmfileConditional       `json:"jxRequirementsIngressExternalDNS,omitempty"`
	IngressTLSCondition         *HelmfileConditional       `json:"jxRequirementsIngressTLS,omitempty"`
	VaultCondition              *HelmfileConditional       `json:"jxRequirementsVault,omitempty"`
	JX                          map[string]interface{}     `json:"jx,omitempty"`
}

// SaveRequirementsValuesFile saves the requirements yaml file for use with helmfile / helm 3
func SaveRequirementsValuesFile(c *jxcore.RequirementsConfig, dir, fileName string) error {
	// lets initialise an empty struct to handle backwards compatibility
	if c.Ingress.TLS == nil {
		c.Ingress.TLS = &jxcore.TLSConfig{}
	}
	jxGlobals, err := loadJXGlobals(dir)
	if err != nil {
		return errors.Wrapf(err, "failed to load global helm values files")
	}

	y := &RequirementsValues{
		RequirementsConfig:          c,
		IngressExternalDNSCondition: &HelmfileConditional{Enabled: c.Ingress.ExternalDNS},
		IngressTLSCondition:         &HelmfileConditional{Enabled: c.Ingress.TLS.Enabled},
		VaultCondition:              &HelmfileConditional{Enabled: c.SecretStorage == jxcore.SecretStorageTypeVault},
		JX:                          jxGlobals,
	}

	err = yamls.SaveFile(y, fileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", fileName)
	}
	log.Logger().Debugf("generated helm YAML file from jx requirements at %s", termcolor.ColorInfo(fileName))
	return nil
}

func loadJXGlobals(dir string) (map[string]interface{}, error) {
	answer := map[string]interface{}{}

	fileNames := []string{
		filepath.Join(dir, "versionStream", "src", "fake-secrets.yaml.gotmpl"),
		filepath.Join(dir, "imagePullSecrets.yaml"),
		filepath.Join(dir, "jx-global-values.yaml"),
	}
	for _, f := range fileNames {
		m := map[string]interface{}{}
		exists, err := files.FileExists(f)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to check if file exists %s", f)
		}
		if exists {
			err := yamls.LoadFile(f, &m)
			if err != nil {
				return nil, errors.Wrapf(err, "failed to load file %s", f)
			}
			maps.CombineMapTrees(answer, m)
		}
	}
	v := answer["jx"]
	m, ok := v.(map[string]interface{})
	if ok {
		return m, nil
	}
	return nil, nil
}
