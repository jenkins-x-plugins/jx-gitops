package scheduler

import (
	"bytes"
	"text/template"

	"github.com/Masterminds/sprig"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/pkg/errors"
)

// EvaluateTemplate evaluates the go template for the given value
func EvaluateTemplate(templateText string, requirements *jxcore.RequirementsConfig) (string, error) {
	if templateText == "" {
		return "", nil
	}
	funcMap := sprig.TxtFuncMap()

	tmpl, err := template.New("value.gotmpl").Option("missingkey=error").Funcs(funcMap).Parse(templateText)
	if err != nil {
		return "", errors.Wrapf(err, "failed to parse template: %s", templateText)
	}

	requirementsMap, err := requirements.ToMap()
	if err != nil {
		return "", errors.Wrapf(err, "failed turn requirements into a map: %v", requirements)
	}

	templateData := map[string]interface{}{
		"Requirements": requirementsMap,
	}

	var buf bytes.Buffer
	err = tmpl.Execute(&buf, templateData)
	if err != nil {
		return "", errors.Wrapf(err, "failed to evaluate template %s", templateText)
	}
	return buf.String(), nil
}
