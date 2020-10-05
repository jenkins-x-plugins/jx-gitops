package hash

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io/ioutil"

	"github.com/jenkins-x/jx-gitops/pkg/cmd/annotate"
	"github.com/jenkins-x/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kyamls"
	"github.com/jenkins-x/jx-helpers/v3/pkg/options"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// DefaultAnnotation the default annotation used for sha256 hashes
const DefaultAnnotation = "jenkins-x.io/hash"

var (
	cmdLong = templates.LongDesc(`
		Annotates the given files with a hash of the given source files for ConfigMaps/Secrets
`)

	cmdExample = templates.Examples(`
		# annotates the Deployments in a dir from some source ConfigMaps
		%s hash -s foo/configmap.yaml -s another/configmap.yaml -d someDir
	`)
)

// AnnotateOptions the options for the command
type Options struct {
	Dir         string
	Annotation  string
	SourceFiles []string
	Filter      kyamls.Filter
}

// NewCmdHashAnnotate creates a command object for the command
func NewCmdHashAnnotate() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "hash",
		Short:   "Annotates the given files with a hash of the given source files for ConfigMaps/Secrets",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringArrayVarP(&o.SourceFiles, "source", "s", nil, "the source files to hash")
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to recursively look for the *.yaml or *.yml files")
	cmd.Flags().StringVarP(&o.Annotation, "annotation", "a", DefaultAnnotation, "the annotation for the hash to add to the files")

	f := &o.Filter
	cmd.Flags().StringArrayVarP(&f.Kinds, "kind", "k", []string{"Deployment"}, "adds Kubernetes resource kinds to filter on to annotate. For kind expressions see: https://github.com/jenkins-x/jx-gitops/tree/master/docs/kind_filters.md")
	cmd.Flags().StringArrayVarP(&f.KindsIgnore, "kind-ignore", "", nil, "adds Kubernetes resource kinds to exclude. For kind expressions see: https://github.com/jenkins-x/jx-gitops/tree/master/docs/kind_filters.md")

	return cmd, o
}

// Run run the command
func (o *Options) Run() error {
	if o.Annotation == "" {
		return options.MissingOption("annotation")

	}
	if len(o.SourceFiles) == 0 {
		return options.MissingOption("source")
	}
	buff := bytes.Buffer{}
	for _, s := range o.SourceFiles {
		exists, err := files.FileExists(s)
		if err != nil {
			return errors.Wrapf(err, "failed to check if file exists %s", s)
		}
		if !exists {
			log.Logger().Warnf("the file to hash %s does not exist so ignoring it from the hash calculation", s)
			continue
		}
		data, err := ioutil.ReadFile(s)
		if err != nil {
			return errors.Wrapf(err, "failed to load source file %s", s)
		}
		_, err = buff.Write(data)
		if err != nil {
			return errors.Wrapf(err, "failed to write data to hash")
		}
	}
	hashBytes := sha256.Sum256(buff.Bytes())
	annotationExpression := fmt.Sprintf("%s=%x", o.Annotation, hashBytes)
	err := annotate.UpdateAnnotateInYamlFiles(o.Dir, []string{annotationExpression}, o.Filter)
	if err != nil {
		return errors.Wrapf(err, "failed to annotate files in dir %s", o.Dir)
	}
	log.Logger().Infof("added annotation: %s to Deployments in dir %s", annotationExpression, o.Dir)
	return nil
}
