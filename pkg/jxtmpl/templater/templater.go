package templater

import (
	"bytes"
	"io/ioutil"
	"text/template"

	"github.com/Masterminds/sprig"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/chartutil"
)

// Templater a templater of values yaml
type Templater struct {
	Requirements *jxcore.RequirementsConfig
	ValuesFiles  []string
}

// NewTemplater creates a new templater
func NewTemplater(requirements *jxcore.RequirementsConfig, valuesFiles []string) *Templater {
	return &Templater{
		Requirements: requirements,
		ValuesFiles:  valuesFiles,
	}
}

func (o *Templater) createFuncMap(requirements *jxcore.RequirementsConfig) (template.FuncMap, error) {
	funcMap := NewFunctionMap()
	return funcMap, nil
}

// Generate generates the destination file from the given source template
func (o *Templater) Generate(sourceFile string, destFile string) error {
	requirements := o.Requirements
	funcMap, err := o.createFuncMap(requirements)
	if err != nil {
		return err
	}

	data, err := o.renderTemplate(sourceFile, funcMap)
	if err != nil {
		return errors.Wrapf(err, "failed to render template file %s", sourceFile)
	}

	err = ioutil.WriteFile(destFile, data, files.DefaultFileWritePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to save output of template %s to %s", sourceFile, destFile)
	}
	return nil
}

// NewFunctionMap creates a new function map for values.tmpl.yaml templating
func NewFunctionMap() template.FuncMap {
	funcMap := sprig.TxtFuncMap()
	funcMap["basicAuth"] = BasicAuth
	funcMap["hashPassword"] = HashPassword
	funcMap["removeScheme"] = RemoveScheme
	return funcMap
}

// RenderTemplate evaluates the given values.yaml file as a go template and returns the output data
func (o *Templater) renderTemplate(templateFile string, funcMap template.FuncMap) ([]byte, error) {
	requirements := o.Requirements
	tmpl, err := template.New("values.yaml.gotmpl").Option("missingkey=error").Funcs(funcMap).ParseFiles(templateFile)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse go template: %s", templateFile)
	}

	requirementsMap, err := requirements.ToMap()
	if err != nil {
		return nil, errors.Wrapf(err, "failed turn requirements into a map: %v", requirements)
	}

	valuesMap := map[string]interface{}{
		"jxRequirements": requirementsMap,
	}
	for _, valuesFile := range o.ValuesFiles {
		values := map[string]interface{}{}
		err = yamls.LoadFile(valuesFile, &values)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to load values file %s", valuesFile)
		}
		for k, v := range values {
			valuesMap[k] = v
		}
	}

	for k, v := range valuesMap {
		log.Logger().Debugf("loaded value %s = %#v", k, v)
	}

	templateData := map[string]interface{}{
		"Values": chartutil.Values(valuesMap),
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, templateData)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to execute Secrets template: %s", templateFile)
	}
	data := buf.Bytes()
	return data, nil
}
