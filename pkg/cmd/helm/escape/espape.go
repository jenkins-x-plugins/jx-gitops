package escape

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

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
	cmdLong = templates.LongDesc(`
		Escapes any {{ or }} characters in the YAML files so they can be included in a helm chart
`)

	cmdExample = templates.Examples(`
		# escapes any yaml files so they can be included in a helm chart 
		%s helm escape --dir myyaml
	`)
)

// Options the options for the command
type Options struct {
	Dir string
}

// NewCmdEscape creates a command object for the command
func NewCmdEscape() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "escape",
		Short:   "Escapes any {{ or }} characters in the YAML files so they can be included in a helm chart",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
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
	return EncodeYAMLFiles(o.Dir)
}

// EncodeYAMLFiles splits any files with multiple resources into separate files
func EncodeYAMLFiles(dir string) error {
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to load file %s", path)
		}

		modified := false
		lines := strings.Split(string(data), "\n")

		buf := strings.Builder{}
		for _, line := range lines {
			encoded := escape(line)
			if encoded != line {
				modified = true
			}
			buf.WriteString(encoded)
			buf.WriteString("\n")
		}

		if !modified {
			return nil
		}

		err = ioutil.WriteFile(path, []byte(buf.String()), files.DefaultFileWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to save %s", path)
		}

		log.Logger().Infof("encoded file %s", termcolor.ColorInfo(path))
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to split YAML files in dir %s", dir)
	}
	return nil
}

const (
	openDelim  = `{{ "{{" }}`
	closeDelim = `{{ "}}" }}`
)

func escape(line string) string {
	i1 := strings.Index(line, "{{")
	i2 := strings.Index(line, "}}")

	if i1 < 0 && i2 < 0 {
		return line
	}
	i := i2
	delim := closeDelim
	if i2 < 0 || (i1 < i2 && i1 >= 0) {
		i = i1
		delim = openDelim
	}
	prefix := line[0:i] + delim
	if i+2 >= len(line) {
		return prefix
	}
	return prefix + escape(line[i+2:])
}
