package split

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/helmhelpers"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	splitLong = templates.LongDesc(`
		Splits any YAML files which define multiple resources into separate files
`)

	splitExample = templates.Examples(`
		# splits any files containing multiple resources
		%s split --dir .
	`)

	// resourcesSeparator is used to separate multiple objects stored in the same YAML file
	resourcesSeparator = "---\n"
)

// Options the options for the command
type Options struct {
	Dir string
}

// NewCmdSplit creates a command object for the command
func NewCmdSplit() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "split",
		Short:   "Splits any YAML files which define multiple resources into separate files",
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
	return ProcessYamlFiles(o.Dir)
}

// ProcessYamlFiles splits any files with multiple resources into separate files
func ProcessYamlFiles(dir string) error {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error { //nolint:staticcheck
		if info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}
		data, err := os.ReadFile(path) //nolint:staticcheck
		if err != nil {
			return errors.Wrapf(err, "failed to load file %s", path)
		}

		input := string(data)
		if strings.HasPrefix(input, resourcesSeparator) {
			input = "\n" + input
		}
		sections := strings.Split(input, "\n"+resourcesSeparator)

		count := 0
		var fileNames []string
		buf := strings.Builder{}
		for _, section := range sections {
			if buf.Len() > 0 {
				buf.WriteString(resourcesSeparator)
			}
			buf.WriteString(section)
			if !helmhelpers.IsWhitespaceOrComments(section) {
				count++

				text := buf.String()
				// remove all newline prefixes
				for {
					if !strings.HasPrefix(text, "\n") {
						break
					}
					text = strings.TrimPrefix(text, "\n")
				}
				fileNames = append(fileNames, text)
				buf.Reset()
			}
		}
		if count >= 1 {
			for i, text := range fileNames {
				name := path
				if i > 0 {
					ex := filepath.Ext(path)
					name = strings.TrimSuffix(path, ex) + strconv.Itoa(i+1) + ex
				}

				// lets remove empty files
				if helmhelpers.IsWhitespaceOrComments(text) {
					// lets remove the file if it exists
					exists, err := files.FileExists(path)
					if err != nil {
						return errors.Wrapf(err, "failed to check if file exists %s", path)
					}
					if exists {
						err = os.Remove(path)
						if err != nil {
							return errors.Wrapf(err, "failed to remove empty file %s", path)
						}
						log.Logger().Infof("removed empty file %s", termcolor.ColorInfo(path))
					}
					continue
				}
				err = os.WriteFile(name, []byte(text), files.DefaultFileWritePermissions)
				if err != nil {
					return errors.Wrapf(err, "failed to save %s", name)
				}
			}
		} else {
			// lets remove the file if it exists
			exists, err := files.FileExists(path)
			if err != nil {
				return errors.Wrapf(err, "failed to check if file exists %s", path)
			}
			if exists {
				err = os.Remove(path)
				if err != nil {
					return errors.Wrapf(err, "failed to remove empty file %s", path)
				}
				log.Logger().Infof("removed empty file %s", termcolor.ColorInfo(path))
			}
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to split YAML files in dir %s", dir)
	}
	return nil
}
