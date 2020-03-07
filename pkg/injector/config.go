package injector

import (
	"github.com/golang/glog"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	tlsCertFileKey = "cert.pem"
	tlsKeyFileKey  = "key.pem"
)

func (wh *Webhook) createEnvoyTLSSecret(name string, namespace string, tlsCert, tlsKey []byte) (*corev1.Secret, error) {
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Data: map[string][]byte{
			tlsCertFileKey: tlsCert,
			tlsKeyFileKey:  tlsKey,
		},
	}
	glog.Infof("Cert: %s", string(tlsCert[:]))
	glog.Infof("Key: %s", string(tlsKey[:]))

	if existing, err := wh.kubeClient.CoreV1().Secrets(namespace).Get(name, metav1.GetOptions{}); err == nil {
		glog.Infof("Updating secret: name=%s, namespace=%s", name, namespace)
		existing.Data = secret.Data
		return wh.kubeClient.CoreV1().Secrets(namespace).Update(existing)
	}

	glog.Infof("Creating secret: name=%s, namespace=%s", name, namespace)
	return wh.kubeClient.CoreV1().Secrets(namespace).Create(secret)
}
