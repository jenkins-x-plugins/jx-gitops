package merge

import (
	"context"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/imdario/mergo"
	"github.com/jenkins-x-plugins/jx-gitops/pkg/rootcmd"
	jxcore "github.com/jenkins-x/jx-api/v4/pkg/apis/core/v4beta1"
	"github.com/jenkins-x/jx-helpers/v3/pkg/cobras/templates"
	"github.com/jenkins-x/jx-helpers/v3/pkg/files"
	"github.com/jenkins-x/jx-helpers/v3/pkg/kube"
	"github.com/jenkins-x/jx-helpers/v3/pkg/termcolor"
	"github.com/jenkins-x/jx-logging/v3/pkg/log"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

const (
	// ConfigMapNamespace the default namespace to look for the ConfigMap for the requirements when not using file mode
	ConfigMapNamespace = "default"

	// ConfigMapName the default name of the ConfigMap for the requirements when not using file mode
	ConfigMapName = "terraform-jx-requirements"

	// ConfigMapKey the data key (entry) in the ConfigMap for the requirements when not using file mode
	ConfigMapKey = "jx-requirements.yml"
)

var (
	cmdLong = templates.LongDesc(`
		Merges values from the given file to the local jx-requirements.yml file

This lets you take requirements from, say, the output of a terraform plan and merge with any other changes inside your GitOps repository
`)

	cmdExample = templates.Examples(`
		# merge requirements from a file
		%s requirements merge -f /tmp/jx-requirements.yml

		# merge requirements from a ConfigMap called 'terraform-jx-requirements' in the default namespace
		%s requirements merge 
	`)
)

// Options the options for the command
type Options struct {
	Dir                  string
	File                 string
	Namespace            string
	ConfigMapName        string
	KubeClient           kubernetes.Interface
	requirements         *jxcore.RequirementsConfig
	requirementsFileName string
}

// NewCmdRequirementsResolve creates a command object for the command
func NewCmdRequirementsMerge() (*cobra.Command, *Options) {
	o := &Options{}

	cmd := &cobra.Command{
		Use:     "merge",
		Short:   "Merges values from the given file to the local jx-requirements.yml file",
		Long:    cmdLong,
		Example: fmt.Sprintf(cmdExample, rootcmd.BinaryName, rootcmd.BinaryName),
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run()
		},
	}
	cmd.Flags().StringVarP(&o.Dir, "dir", "d", ".", "the source directory to merge changes into")
	cmd.Flags().StringVarP(&o.File, "file", "f", "", "the requirements file to merge into the source directory")
	cmd.Flags().StringVarP(&o.Namespace, "namespace", "", ConfigMapNamespace, "the namespace used to find the ConfigMap if using the ConfigMap mode")
	cmd.Flags().StringVarP(&o.ConfigMapName, "configmap", "c", ConfigMapName, "the name of the ConfigMap to find the requirements to merge if not specifying a requirements file via --file")
	return cmd, o
}

// Run implements the command
func (o *Options) Run() error {
	var err error
	if o.File == "" {
		o.File, err = o.loadRequirementsFileFromConfigMap()
		if err != nil {
			return errors.Wrapf(err, "failed to load the 'jx-requirements.yml' from the ConfigMap")
		}
		if o.File == "" {
			return nil
		}

	}
	var requirementsResource *jxcore.Requirements
	requirementsResource, o.requirementsFileName, err = jxcore.LoadRequirementsConfig(o.Dir, false)
	if err != nil {
		return errors.Wrapf(err, "failed to load requirements in dir %s", o.Dir)
	}
	o.requirements = &requirementsResource.Spec
	if o.requirementsFileName == "" {
		o.requirementsFileName = filepath.Join(o.Dir, jxcore.RequirementsConfigFileName)
	}

	// lets not se the usual loading as we don't want any default values populated
	requirementChanges, err := jxcore.LoadRequirementsConfigFileNoDefaults(o.File, false)
	if err != nil {
		return errors.Wrapf(err, "failed to load requirement changes from file: %s", o.File)
	}
	if requirementChanges != nil {
		// lets remove any environments as those are always defined in the dev cluster git repository
		requirementChanges.Spec.Environments = nil
	}

	exists := false
	if o.requirements != nil {
		exists, err = files.FileExists(o.requirementsFileName)
		if err != nil {
			return errors.Wrapf(err, "failed to check if file exists %s", o.requirementsFileName)
		}
	}

	if exists {
		err = o.MergeChanges(requirementChanges)
		if err != nil {
			return errors.Wrapf(err, "failed to merge changes from %s", o.File)
		}
	} else {
		o.requirements = &requirementChanges.Spec
	}

	err = requirementsResource.SaveConfig(o.requirementsFileName)
	if err != nil {
		return errors.Wrapf(err, "failed to save file %s", o.requirementsFileName)
	}
	log.Logger().Infof("saved file %s", termcolor.ColorInfo(o.requirementsFileName))
	return nil

}

// MergeChanges merges changes from the given requirements into the source
func (o *Options) MergeChanges(reqs *jxcore.Requirements) error {
	err := mergo.Merge(o.requirements, reqs.Spec, mergo.WithOverride)
	if err != nil {
		return errors.Wrap(err, "error merging requirements")
	}

	// if domain is set in terraform always use that to override cluster requirements as domains are managed by Terraform
	if reqs.Spec.Ingress.Domain != "" {
		o.requirements.Ingress.Domain = reqs.Spec.Ingress.Domain
	}

	return nil
}

func (o *Options) loadRequirementsFileFromConfigMap() (string, error) {
	var err error
	o.KubeClient, err = kube.LazyCreateKubeClient(o.KubeClient)
	if err != nil {
		return "", errors.Wrapf(err, "failed to create kube client")
	}

	cm, err := o.KubeClient.CoreV1().ConfigMaps(o.Namespace).Get(context.TODO(), o.ConfigMapName, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Logger().Infof("there is no ConfigMap %s in namespace %s so no need to merge requirements", o.ConfigMapName, o.Namespace)
			return "", nil
		}
		return "", errors.Wrapf(err, "failed to load ConfigMap %s in namespace %s", o.ConfigMapName, o.Namespace)
	}
	text := ""
	if cm.Data != nil {
		text = cm.Data[ConfigMapKey]
	}
	if text == "" {
		log.Logger().Warnf("the ConfigMap %s in namespace %s has no %s entry", o.ConfigMapName, o.Namespace, ConfigMapKey)
		return "", nil
	}

	tmpFile, err := ioutil.TempFile("", "")
	if err != nil {
		return "", errors.Wrapf(err, "failed to create a temporaray file")
	}
	fileName := tmpFile.Name()
	err = ioutil.WriteFile(fileName, []byte(text), files.DefaultFileWritePermissions)
	if err != nil {
		return "", errors.Wrapf(err, "failed to write jx-requirements.yml file to %s", fileName)
	}
	log.Logger().Infof("wrote the ConfigMap jx-requirements.yml to %s", fileName)
	return fileName, nil
}
