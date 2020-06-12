package label

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/common"
	"github.com/jenkins-x/jx-gitops/pkg/mapslices"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"
)

var (
	labelLong = templates.LongDesc(`
		Updates all kubernetes resources in the given directory tree to add/override the given label
`)

	labelExample = templates.Examples(`
		# updates recursively all resources 
		%s step update label --dir . foo=bar
	`)
)

// LabelOptions the options for the command
type Options struct {
	Dir   string
	Label string
}

// NewCmdUpdate creates a command object for the command
func NewCmdUpdateLabel() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "label",
		Short:   "Updates all kubernetes resources in the given directory tree to add/override the given label",
		Long:    labelLong,
		Example: fmt.Sprintf(labelExample, common.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := UpdateLabelArgsInYamlFiles(o.Dir, args)
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "", ".", "the directory to recursively look for the *.yaml or *.yml files")

	return cmd, o
}

func UpdateLabelArgsInYamlFiles(dir string, args []string) error {
	m := toMap(args)
	return UpdateLabelInYamlFiles(dir, m)
}

func toMap(args []string) map[string]string {
	m := map[string]string{}
	for _, a := range args {
		paths := strings.SplitN(a, "=", 2)
		k := paths[0]
		v := ""
		if len(paths) > 1 {
			v = paths[1]
		}
		m[k] = v
	}
	return m
}

// UpdateLabelInYamlFiles updates the labels in yaml files
func UpdateLabelInYamlFiles(dir string, labels map[string]string) error {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		// lets load the file
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to load file %s", path)
		}

		ms := yaml.MapSlice{}
		err = yaml.Unmarshal(data, &ms)
		if err != nil {
			return errors.Wrapf(err, "failed to unmarshal YAML file %s", path)
		}

		modified := false
		flag := false
		for k, v := range labels {
			fields := []string{"metadata", "labels", k}
			ms, flag, err = mapslices.SetNestedField(ms, v, fields...)
			if err != nil {
				return errors.Wrapf(err, "failed to set fields %#v to value %v", fields, v)
			}
			if flag {
				modified = true
			}
		}
		if !modified {
			return nil
		}
		data, err = yaml.Marshal(ms)
		if err != nil {
			return errors.Wrapf(err, "failed to marshal modified file %s", path)
		}

		err = ioutil.WriteFile(path, data, util.DefaultFileWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to save %s", path)
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to set labels %#v in dir %s", labels, dir)
	}
	return nil
}
