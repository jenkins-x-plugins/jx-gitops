package ingress

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/jenkins-x/jx-gitops/pkg/common"
	"github.com/jenkins-x/jx/pkg/cmd/helper"
	"github.com/jenkins-x/jx/pkg/cmd/templates"
	"github.com/jenkins-x/jx/pkg/config"
	"github.com/jenkins-x/jx/pkg/gits"
	"github.com/jenkins-x/jx/pkg/log"
	"github.com/jenkins-x/jx/pkg/util"
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
		%s step update ingress
	`)
)

// IngressOptions the options for the command
type Options struct {
	Dir           string
	ReplaceDomain string
	Gitter        gits.Gitter
	BatchMode     bool
}

// NewCmdUpdate creates a command object for the command
func NewCmdUpdateIngress() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "ingress",
		Short:   "Updates Ingress resources with the current ingress domain",
		Long:    ingressLong,
		Example: fmt.Sprintf(ingressExample, common.BinaryName),
		Run: func(cmd *cobra.Command, args []string) {
			err := o.Run()
			helper.CheckErr(err)
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the directory to look for a 'jx-apps.yml' file")
	cmd.Flags().StringVarP(&o.ReplaceDomain, "domain", "n", "cluster.local", "the domain to replace with whats in jx-requirements.yml")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	requirements, requirementsFileName, err := config.LoadRequirementsConfig(o.Dir)
	if err != nil {
		return errors.Wrapf(err, "failed to load requirements in dir %s", o.Dir)
	}

	newDomain := requirements.Ingress.Domain
	if newDomain == "" {
		log.Logger().Warnf("not modifying Ingress resources as the requirements file %s does not contain Ingress.Domain", requirementsFileName)
		return nil
	}
	tlsEnabled := requirements.Ingress.TLS.Enabled

	log.Logger().Infof("replacing ingress domain %s to %s with TLS: %v", util.ColorInfo(o.ReplaceDomain), util.ColorInfo(newDomain), tlsEnabled)

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
				log.Logger().Infof("ingress at %s updated to %s", util.ColorInfo(path), util.ColorInfo(host))
			} else {
				log.Logger().Infof("ingress at %s does not match domain as is %s", util.ColorInfo(path), util.ColorInfo(currentHost))
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
						log.Logger().Infof("ingress at %s disabling TLS", util.ColorInfo(path))
						s.TLS = nil
						break
					}
					log.Logger().Infof("ingress at %s updated to %s", util.ColorInfo(path), util.ColorInfo(host))
					hosts[j] = host
					s.TLS[i].Hosts = hosts
				} else {
					log.Logger().Infof("ingress at %s does not match domain as is %s", util.ColorInfo(path), util.ColorInfo(currentHost))
				}
			}
		}
		return modified, nil
	}
	return UpdateIngresses(o.Dir, fn)
}

// modifyHost modifies the host name if it matches the predicate otherwise return an empty string
func (o *Options) modifyHost(host string, newDomain string) (string, error) {
	if strings.HasSuffix(host, o.ReplaceDomain) {
		return strings.TrimSuffix(host, o.ReplaceDomain) + newDomain, nil
	}
	return "", nil
}

func UpdateIngresses(dir string, fn func(ing *v1beta1.Ingress, path string) (bool, error)) error {
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
			return errors.Wrapf(err, "failed to unmarshal YAML in file %s", path)
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
		err = ioutil.WriteFile(path, data, util.DefaultFileWritePermissions)
		if err != nil {
			return errors.Wrapf(err, "failed to save %s", path)
		}
		return nil
	})
}
