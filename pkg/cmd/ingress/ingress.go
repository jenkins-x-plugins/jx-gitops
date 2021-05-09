package ingress

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/helper"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/api/extensions/v1beta1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/yaml"
)

var (
	ingressLong = templates.LongDesc(`
		Updates Ingress resources with the current ingress domain 
`)

	ingressExample = templates.Examples(`
		# updates any newly created Ingress resources to the new domain
		%s ingress
	`)
)

// IngressOptions the options for the command
type Options struct {
	Dir                  string
	ReplaceDomain        string
	BatchMode            bool
	FailOnYAMLParseError bool
}

// NewCmdUpdate creates a command object for the command
func NewCmdUpdateIngress() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "ingress",
		Short:   "Updates Ingress resources with the current ingress domain",
		Long:    ingressLong,
		Example: fmt.Sprintf(ingressExample, rootcmd.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to look for a 'jx-apps.yml' file")
	cmd.Flags().StringVarP(&o.ReplaceDomain, "domain", "n", "cluster.local", "the domain to replace with whats in jx-requirements.yml")
	cmd.Flags().BoolVarP(&o.FailOnYAMLParseError, "fail-on-parse-error", "", false, "if enabled we fail if we cannot parse a yaml file as a kubernetes resource")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	requirementsResource, requirementsFileName, err := jxcore.LoadRequirementsConfig(o.Dir, false)
	if err != nil {
		return errors.Wrapf(err, "failed to load requirements in dir %s", o.Dir)
	}
	requirements := &requirementsResource.Spec
	newDomain := requirements.Ingress.Domain
	if newDomain == "" {
		log.Logger().Warnf("not modifying Ingress resources as the requirements file %s does not contain Ingress.Domain", requirementsFileName)
		return nil
	}
	tlsEnabled := requirements.Ingress.TLS.Enabled

	log.Logger().Infof("replacing ingress domain %s to %s with TLS: %v", termcolor.ColorInfo(o.ReplaceDomain), termcolor.ColorInfo(newDomain), tlsEnabled)

	fn := func(ing *v1beta1.Ingress, path string) (bool, error) {
		modified := false
		s := &ing.Spec
		for i, r := range s.Rules {
			currentHost := r.Host
			host, err := o.modifyHost(currentHost, newDomain)
			if err != nil {
				return modified, err
			}
			if host != "" && host != currentHost {
				modified = true
				s.Rules[i].Host = host
				log.Logger().Infof("ingress at %s updated to %s", termcolor.ColorInfo(path), termcolor.ColorInfo(host))
			} else {
				log.Logger().Infof("ingress at %s does not match domain as is %s", termcolor.ColorInfo(path), termcolor.ColorInfo(currentHost))
			}
		}
		for i, tls := range s.TLS {
			hosts := tls.Hosts
			for j, currentHost := range hosts {
				host, err := o.modifyHost(currentHost, newDomain)
				if err != nil {
					return modified, err
				}
				if host != "" && host != currentHost {
					modified = true
					if !tlsEnabled {
						log.Logger().Infof("ingress at %s disabling TLS", termcolor.ColorInfo(path))
						s.TLS = nil
						break
					}
					log.Logger().Infof("ingress at %s updated to %s", termcolor.ColorInfo(path), termcolor.ColorInfo(host))
					hosts[j] = host
					s.TLS[i].Hosts = hosts
				} else {
					log.Logger().Infof("ingress at %s does not match domain as is %s", termcolor.ColorInfo(path), termcolor.ColorInfo(currentHost))
				}
			}
		}
		return modified, nil
	}
	return o.updateIngresses(o.Dir, fn)
}

// modifyHost modifies the host name if it matches the predicate otherwise return an empty string
func (o *Options) modifyHost(host string, newDomain string) (string, error) {
	if strings.HasSuffix(host, o.ReplaceDomain) {
		return strings.TrimSuffix(host, o.ReplaceDomain) + newDomain, nil
	}
	return "", nil
}

func (o *Options) updateIngresses(dir string, fn func(ing *v1beta1.Ingress, path string) (bool, error)) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if info == nil || info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(path, ".yaml") && !strings.HasSuffix(path, ".yml") {
			return nil
		}

		// ignore some files
		_, fileName := filepath.Split(path)
		if fileName == "jx-requirements.yml" || fileName == "jenkins-x.yml" {
			return nil
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			return errors.Wrapf(err, "failed to load file %s", path)
		}

		obj := &unstructured.Unstructured{}
		err = yaml.Unmarshal(data, obj)
		if err != nil {
			if o.FailOnYAMLParseError {
				return errors.Wrapf(err, "failed to unmarshal YAML in file %s", path)
			}
			log.Logger().Infof("could not parse YAML file %s", path)
			return nil
		}
		if obj.GetKind() != "Ingress" {
			return nil
		}
		apiVersion := obj.GetAPIVersion()
		if apiVersion != "networking.k8s.io/v1beta1" && apiVersion != "extensions/v1beta1" {
			return nil
		}

		ing := &v1beta1.Ingress{}
		err = yaml.Unmarshal(data, ing)
		if err != nil {
			return errors.Wrapf(err, "failed to unmarshal YAML as Ingress in file %s", path)
		}

		modified, err := fn(ing, path)
		if err != nil {
			return errors.Wrapf(err, "failed to modify Ingress at %s", path)
		}
		if !modified {
			return nil
		}
		data, err = yaml.Marshal(ing)
		err = ioutil.WriteFile(path, data, files.DefaultFileWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to save %s", path)
		}
		return nil
	})
}
