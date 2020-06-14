package split

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/common"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/util"
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
	resourcesSeparator = "---"
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
		Example: fmt.Sprintf(splitExample, common.BinaryName),
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
	return SplitYamlFiles(o.Dir)
}

// SplitYamlFiles splits any files with multiple resources into separate files
func SplitYamlFiles(dir string) error {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to load file %s", path)
		}

		sections := strings.Split(string(data), resourcesSeparator)

		count := 0
		var files []string
		buf := strings.Builder{}
		for _, section := range sections {
			if buf.Len() > 0 {
				buf.WriteString("\n")
				buf.WriteString(resourcesSeparator)
				buf.WriteString("\n")
			}
			buf.WriteString(section)
			if !isWhitespaceOrComments(section) {
				count++

				text := buf.String()
				// remove all newline prefixes
				for {
					if !strings.HasPrefix(text, "\n") {
						break
					}
					text = strings.TrimPrefix(text, "\n")
				}
				files = append(files, text)
				buf.Reset()
			}
		}
		if count > 1 {
			for i, text := range files {
				name := path
				if i > 0 {
					ex := filepath.Ext(path)
					name = strings.TrimSuffix(path, ex) + strconv.Itoa(i+1) + ex
				}
				err = ioutil.WriteFile(name, []byte(text), util.DefaultFileWritePermissions)
				if err != nil {
					return errors.Wrapf(err, "failed to save %s", name)
				}
			}
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to split YAML files in dir %s", dir)
	}
	return nil
}

// isWhitespaceOrComments returns true if the text is empty, whitespace or comments only
func isWhitespaceOrComments(text string) bool {
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		t := strings.TrimSpace(line)
		if t != "" && !strings.HasPrefix(t, "#") {
			return false
		}
	}
	return true
}
