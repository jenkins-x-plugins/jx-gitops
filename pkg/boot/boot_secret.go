package boot

import (
	"github.com/jenkins-x/jx-logging/pkg/log"
	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// BootSecret loads the boot secret
type BootSecret struct {
	URL      string
	Username string
	Password string
}

// LoadBootSecret loads the boot secret from the current namespace
func LoadBootSecret(kubeClient kubernetes.Interface, ns, operatorNamespace, secretName, defaultUserName string) (*BootSecret, error) {
	secret, err := kubeClient.CoreV1().Secrets(ns).Get(secretName, metav1.GetOptions{})
	if err != nil && operatorNamespace != ns {
		var err2 error
		secret, err2 = kubeClient.CoreV1().Secrets(operatorNamespace).Get(secretName, metav1.GetOptions{})
		if err2 == nil {
			err = nil
		}
	}
	if err != nil {
		if !apierrors.IsNotFound(err) {
			log.Logger().Warnf("could not find secret %s in namespace %s", secretName, ns)
			return nil, nil
		}
		return nil, errors.Wrapf(err, "failed to find Secret %s in namespace %s", secretName, ns)
	}
	answer := &BootSecret{}
	data := secret.Data
	if data != nil {
		answer.URL = string(data["url"])
		if answer.URL == "" {
			log.Logger().Warnf("secret %s in namespace %s does not have a url entry", secretName, ns)
		}
		answer.Username = string(data["username"])
		if answer.Username == "" {
			answer.Username = defaultUserName
		}
		answer.Password = string(data["password"])
	}
	return answer, nil
}
