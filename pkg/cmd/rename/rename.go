package rename

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/yaml"
)

var (
	splitLong = templates.LongDesc(`
		Renames yaml files to use canonical file names based on the resource name and kind
`)

	splitExample = templates.Examples(`
		# renames files to use a canonical file name
		%s rename --dir .
	`)

	// resourcesSeparator is used to separate multiple objects stored in the same YAML file
	resourcesSeparator = "---\n"
)

// Options the options for the command
type Options struct {
	Dir     string
	Verbose bool
}

// NewCmdRename creates a command object for the command
func NewCmdRename() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "rename",
		Short:   "Renames yaml files to use canonical file names based on the resource name and kind",
		Long:    splitLong,
		Example: fmt.Sprintf(splitExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to recursively look for the *.yaml or *.yml files")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	err := filepath.Walk(o.Dir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		b, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to load file %s", path)
		}
		safeYaml := RemoveGoTemplateLines(b)
		node, err := yaml.Parse(safeYaml)
		if err != nil {
			return errors.Wrapf(err, "failed to load file %s", path)
		}

		name := kyamls.GetName(node, path)
		if name == "" {
			log.Logger().Warnf("no name for file %s so ignoring", path)
			return nil
		}

		kind := kyamls.GetKind(node, path)
		apiVersion := kyamls.GetAPIVersion(node, path)

		dir, file := filepath.Split(path)
		ext := filepath.Ext(path)

		cn := o.canonicalName(apiVersion, kind, name)

		newFile := cn + ext
		newPath := filepath.Join(dir, newFile)

		if newPath != path {
			if o.Verbose {
				log.Logger().Infof("renaming %s => %s", file, newFile)
			} else {
				log.Logger().Debugf("renaming %s => %s", file, newFile)
			}
			err = os.Rename(path, newPath)
			if err != nil {
				return errors.Wrapf(err, "failed to rename %s to %s", file, newFile)
			}

		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to rename YAML files in dir %s", o.Dir)
	}
	return nil
}

// RemoveGoTemplateLines removes any lines which start with go templates so that we can parse as much of the
// YAML as possible; such as resources with some templating inside the spec
func RemoveGoTemplateLines(b []byte) string {
	lines := strings.Split(string(b), "\n")

	buf := &strings.Builder{}
	for _, line := range lines {
		t := strings.TrimSpace(line)
		// ignore go templates
		if strings.HasPrefix(t, "{{") {
			continue
		}
		buf.WriteString(line)
		buf.WriteString("\n")
	}
	return buf.String()
}

var (
	kindSuffixes = map[string]string{
		"clusterrolebinding":             "crb",
		"configmap":                      "cm",
		"customresourcedefinition":       "crd",
		"deployment":                     "deploy",
		"mutatingwebhookconfiguration":   "mutwebhookcfg",
		"namespace":                      "ns",
		"rolebinding":                    "rb",
		"service":                        "svc",
		"serviceaccount":                 "sa",
		"validatingwebhookconfiguration": "valwebhookcfg",
	}
)

func (o *Options) canonicalName(apiVersion, kind, name string) string {
	lk := strings.ToLower(kind)
	suffix := kindSuffixes[lk]
	if suffix == "svc" && strings.Contains(apiVersion, "knative") {
		suffix = "ksvc"
	}
	if suffix == "" {
		suffix = lk
	}
	// lets replace any odd characters
	name = strings.ReplaceAll(name, ":", "-")
	name = strings.ReplaceAll(name, string(os.PathSeparator), "-")
	if kind == "" {
		return name
	}
	return name + "-" + suffix
}
