package build

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/chart"
	"github.com/jenkins-x/jx-helpers/v3/pkg/yamls"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/plugins"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cmdrunner"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

var (
	info = termcolor.ColorInfo

	cmdLong = templates.LongDesc(`
		Builds and lints any helm charts
`)

	cmdExample = templates.Examples(`
		# generates the resources from a helm chart
		%s step helm template
	`)
)

// Options the options for the command
type Options struct {
	UseHelmPlugin      bool
	HelmBinary         string
	ChartsDir          string
	OCI                bool
	RegistryConfigFile string
	RepositoryUsername string
	RepositoryPassword string
	RepositoryURL      string
	CommandRunner      cmdrunner.CommandRunner
}

// NewCmdHelmBuild creates a command object for the command
func NewCmdHelmBuild() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "build",
		Short:   "Builds and lints any helm charts",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName),
		Run: func(_ *cobra.Command, _ []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.ChartsDir, "charts-dir", "c", "charts", "the directory to look for helm charts to release")
	cmd.Flags().StringVarP(&o.HelmBinary, "binary", "n", "", "specifies the helm binary location to use. If not specified defaults to 'helm' on the $PATH")
	cmd.Flags().BoolVarP(&o.UseHelmPlugin, "use-helm-plugin", "", false, "uses the jx binary plugin for helm rather than whatever helm is on the $PATH")
	cmd.Flags().StringVarP(&o.RepositoryUsername, "repo-username", "", "", "the username to access the chart repository. If not specified defaults to the environment variable $JX_REPOSITORY_USERNAME")
	cmd.Flags().StringVarP(&o.RepositoryPassword, "repo-password", "", "", "the password to access the chart repository. If not specified defaults to the environment variable $JX_REPOSITORY_PASSWORD")
	cmd.Flags().BoolVarP(&o.OCI, "oci", "", false, "using OCI charts")
	cmd.Flags().StringVarP(&o.RegistryConfigFile, "registry-config", "", "/tekton/creds-secrets/tekton-container-registry-auth/.dockerconfigjson", "the path to the registry config for OCI login")
	return cmd, o
}

// Run implements the command
func (o *Options) Validate() error {
	if o.CommandRunner == nil {
		o.CommandRunner = cmdrunner.DefaultCommandRunner
	}
	var err error
	if o.HelmBinary == "" {
		if o.UseHelmPlugin {
			o.HelmBinary, err = plugins.GetHelmBinary(plugins.HelmVersion)
			if err != nil {
				return err
			}
		}
		if o.HelmBinary == "" {
			o.HelmBinary = "helm"
		}
	}
	if o.RepositoryUsername == "" {
		o.RepositoryUsername = os.Getenv("JX_REPOSITORY_USERNAME")
		if o.RepositoryUsername == "" {
			o.RepositoryUsername = os.Getenv("GITHUB_REPOSITORY_OWNER")
		}
		if o.RepositoryUsername == "" {
			o.RepositoryUsername = os.Getenv("GIT_USERNAME")
		}
		if o.RepositoryUsername == "" {
			o.RepositoryUsername = os.Getenv("GITHUB_ACTOR")
		}
	}
	if o.RepositoryPassword == "" {
		o.RepositoryPassword = os.Getenv("JX_REPOSITORY_PASSWORD")
		if o.RepositoryPassword == "" {
			o.RepositoryPassword = os.Getenv("GITHUB_TOKEN")
		}
	}

	return nil
}

// Run implements the command
func (o *Options) Run() error {
	err := o.Validate()
	if err != nil {
		return errors.Wrapf(err, "failed to validate")
	}
	dir := o.ChartsDir
	exists, err := files.DirExists(dir)
	if err != nil {
		return errors.Wrapf(err, "failed to check if charts dir exists %s", dir)
	}
	if !exists {
		log.Logger().Infof("no charts dir: %s", dir)
		return nil
	}

	fileSlice, err := os.ReadDir(dir)
	if err != nil {
		return errors.Wrapf(err, "failed to read dir %s", dir)
	}
	count := 0
	for _, f := range fileSlice {
		if !f.IsDir() {
			continue
		}
		name := f.Name()
		chartDir := filepath.Join(dir, name)
		chartFile := filepath.Join(chartDir, "Chart.yaml")
		exists, err := files.FileExists(chartFile)
		if err != nil {
			return errors.Wrapf(err, "failed to check file exists %s", chartFile)
		}
		if !exists {
			continue
		}

		chartDef := &chart.Chart{}
		if exists {
			err = yamls.LoadFile(chartFile, chartDef)
			if err != nil {
				return errors.Wrapf(err, "failed to load Chart.yaml")
			}

			for i, dependency := range chartDef.Dependencies {
				log.Logger().Infof("Adding repository for dependency %s", dependency.Name)
				if dependency.Repository != "" && !strings.HasPrefix(dependency.Repository, "oci://") {
					c := &cmdrunner.Command{
						Dir:  chartDir,
						Name: o.HelmBinary,
						Args: []string{"repo", "add", strconv.Itoa(i), dependency.Repository},
					}
					_, err = o.CommandRunner(c)
					if err != nil {
						return errors.Wrapf(err, "failed to add repository")
					}
				} else {
					log.Logger().Infof("Skipping local dependency %s", dependency.Name)
				}
			}
		}

		log.Logger().Infof("building chart %s", info(name))

		c := &cmdrunner.Command{
			Dir:  chartDir,
			Name: o.HelmBinary,
			Args: []string{"lint"},
		}
		_, err = o.CommandRunner(c)
		if err != nil {
			return errors.Wrapf(err, "failed to lint")
		}
		if o.OCI {
			if o.RepositoryPassword == "" {
				log.Logger().Debugf("OCI helm dependency build using --registry-config  %s", info(o.RegistryConfigFile))
				c = &cmdrunner.Command{
					Dir:  chartDir,
					Name: o.HelmBinary,
					Args: []string{"dependency", "build", ".", "--registry-config", o.RegistryConfigFile},
				}
				_, err = o.CommandRunner(c)
				if err != nil {
					return errors.Wrapf(err, "failed to build dependencies")
				}
			} else {
				log.Logger().Debugf("OCI helm dependency build using username/password '%s/***'", info(o.RepositoryUsername))

				c = &cmdrunner.Command{
					Dir:  chartDir,
					Name: o.HelmBinary,
					Args: []string{"registry", "login", o.RepositoryURL, "--username", o.RepositoryUsername, "--password", o.RepositoryPassword},
				}
				_, err = o.CommandRunner(c)
				if err != nil {
					return errors.Wrapf(err, "failed to helm login")
				}
				c = &cmdrunner.Command{
					Dir:  chartDir,
					Name: o.HelmBinary,
					Args: []string{"dependency", "build", "."},
				}
				_, err = o.CommandRunner(c)
				if err != nil {
					return errors.Wrapf(err, "failed to build dependencies")
				}
			}
		} else {
			c = &cmdrunner.Command{
				Dir:  chartDir,
				Name: o.HelmBinary,
				Args: []string{"dependency", "build", "."},
			}
			_, err = o.CommandRunner(c)
			if err != nil {
				return errors.Wrapf(err, "failed to build dependencies")
			}
		}
		if o.OCI && o.RepositoryPassword == "" {
			c = &cmdrunner.Command{
				Dir:  chartDir,
				Name: o.HelmBinary,
				Args: []string{"package", ".", "--registry-config", o.RegistryConfigFile},
			}
		} else {
			c = &cmdrunner.Command{
				Dir:  chartDir,
				Name: o.HelmBinary,
				Args: []string{"package", "."},
			}
		}
		_, err = o.CommandRunner(c)
		if err != nil {
			return errors.Wrapf(err, "failed to package")
		}
		count++
	}

	log.Logger().Infof("built %d charts from the charts dir: %s", count, dir)
	return nil
}
