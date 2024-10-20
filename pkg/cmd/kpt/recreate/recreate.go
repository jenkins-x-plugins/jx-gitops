package recreate

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

var (
	kptLong = templates.LongDesc(`
		Updates the kpt packages in the given directory
`)

	kptExample = templates.Examples(`
		# updates the kpt of all the yaml resources in the given directory
		%s kpt --dir .
	`)

	pathSeparator = string(os.PathSeparator)
)

// Options the options for the command
type Options struct {
	Dir           string
	OutDir        string
	Version       string
	IgnoreErrors  bool
	DryRun        bool
	CommandRunner cmdrunner.CommandRunner
}

// NewCmdKptRecreate creates a command object for the command
func NewCmdKptRecreate() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "recreate",
		Short:   "Recreates the kpt packages in the given directory",
		Long:    kptLong,
		Example: fmt.Sprintf(kptExample, rootcmd.BinaryName),
		Run: func(_ *cobra.Command, _ []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to recursively look for the *.yaml or *.yml files")
	cmd.Flags().StringVarP(&o.OutDir, "out-dir", "o", "", "the output directory to generate the output")
	cmd.Flags().StringVarP(&o.Version, "version", "", "", "if specified overrides the versions used in the kpt packages (e.g. to 'master')")
	cmd.Flags().BoolVarP(&o.IgnoreErrors, "ignore-errors", "i", false, "if enabled we continue processing on kpt errors")
	cmd.Flags().BoolVarP(&o.DryRun, "dry-run", "", false, "just output the commands to be executed")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	if o.Dir == "" {
		o.Dir = "."
	}
	dir, err := filepath.Abs(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to find abs dir of %s", o.Dir)
	}

	if o.OutDir == "" {
		o.OutDir, err = os.MkdirTemp("", "")
		if err != nil {
			return errors.Wrap(err, "failed to create temp dir")
		}
	}
	if o.DryRun {
		o.CommandRunner = cmdrunner.DryRunCommandRunner
	}
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
	}

	err = files.CopyDirOverwrite(dir, o.OutDir)
	if err != nil {
		return errors.Wrapf(err, "failed to copy %s to %s", dir, o.OutDir)
	}
	dir = o.OutDir

	err = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error { //nolint:staticcheck
		if info == nil || info.IsDir() {
			return nil
		}
		kptDir, name := filepath.Split(path)
		if name != "Kptfile" {
			return nil
		}
		rel, err := filepath.Rel(dir, kptDir) //nolint:staticcheck
		if err != nil {
			return errors.Wrapf(err, "failed to calculate the relative directory of %s", kptDir)
		}
		kptDir = strings.TrimSuffix(kptDir, pathSeparator)
		_, kptDirName := filepath.Split(kptDir)

		u := &unstructured.Unstructured{}
		data, err := os.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to read file %s", path)
		}
		err = yaml.Unmarshal(data, u)
		if err != nil {
			return errors.Wrapf(err, "failed to parse URL for %s", path)
		}

		gitURL, _, err := unstructured.NestedString(u.Object, "upstream", "git", "repo")
		if err != nil {
			return errors.Wrapf(err, "failed to find git URL for path %s", path)
		}
		if gitURL == "" {
			return errors.Errorf("no git URL for path %s", path)
		}
		directory, _, err := unstructured.NestedString(u.Object, "upstream", "git", "directory")
		if err != nil {
			return errors.Wrapf(err, "failed to find git directory for path %s", path)
		}
		if directory == "" {
			return errors.Errorf("no git directory for path %s", path)
		}
		version := o.Version
		if version == "" {
			version, _, err = unstructured.NestedString(u.Object, "upstream", "git", "commit")
			if err != nil {
				return errors.Wrapf(err, "failed to find git commit for path %s", path)
			}
			if version == "" {
				return errors.Errorf("no git version for path %s", path)
			}
		}

		if !strings.HasSuffix(gitURL, ".git") {
			gitURL = strings.TrimSuffix(gitURL, "/") + ".git"
		}
		if !strings.HasPrefix(directory, pathSeparator) {
			directory = pathSeparator + directory
		}

		expression := fmt.Sprintf("%s%s@%s", gitURL, directory, version)
		directories := strings.Split(directory, pathSeparator)

		// if the folder resource name is the same as the namespace then lets omit
		destDir := rel
		if directories[len(directories)-1] == kptDirName {
			destDir, _ = filepath.Split(rel)
			destDir = strings.TrimSuffix(destDir, pathSeparator)
		}
		args := []string{"pkg", "get", expression, destDir}
		c := &cmdrunner.Command{
			Name: "kpt",
			Args: args,
			Dir:  dir,
		}

		err = os.RemoveAll(kptDir)
		if err != nil {
			return errors.Wrapf(err, "failed to remove kpt directory %s", kptDir)
		}
		text, err := o.CommandRunner(c)
		log.Logger().Info(text)
		if err != nil {
			if !o.IgnoreErrors {
				return errors.Wrapf(err, "failed to run kpt command")
			}
			log.Logger().Warn(err.Error())
		}
		return nil
	})
	if err != nil {
		return errors.Wrapf(err, "failed to upgrade kpt packages in dir %s", dir)
	}
	return nil
}
