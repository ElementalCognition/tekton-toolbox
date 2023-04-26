package clusterinterceptorupdater

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"knative.dev/pkg/webhook/certificates/resources"

	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	clusterinterceptorsinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/clusterinterceptor"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	corev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
)

func GetNamespace() string {
	if ns := os.Getenv("SVC_NAMESPACE"); ns != "" {
		return ns
	}
	return "tekton-pipelines"
}

// Returns key, cert, caCert and error.
func generateCertificates(ctx context.Context, svc, ns string) ([]byte, []byte, []byte, error) {
	if svc == "" || ns == "" {
		return nil, nil, nil, errors.New("serviceName and or namespace is empty")
	}
	expiration := time.Now().AddDate(10, 0, 0)
	fmt.Printf("Generate certificates for svc %s in %s namespace.\n", svc, ns)
	key, cert, caCert, err := resources.CreateCerts(ctx, svc, ns, expiration)
	if err != nil {
		return nil, nil, nil, err
	}
	return key, cert, caCert, nil
}

func getCreateCertsSecret(ctx context.Context, coreV1Interface corev1.CoreV1Interface,
	log *zap.SugaredLogger, sn, ns string) (*v1.Secret, error) {
	s, err := coreV1Interface.Secrets(ns).Get(ctx, sn, metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			log.Infof("secret %s is missing, creating", sn)
			key, cert, cacert, err := generateCertificates(ctx, sn, ns)
			if err != nil {
				log.Infof("Failed to generate certificates", zap.Error(err))
				return nil, err
			}
			secret := &v1.Secret{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Secret",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      sn,
					Namespace: ns,
				},
				Data: map[string][]byte{
					"ca-cert.pem":     cacert,
					"server-cert.pem": cert,
					"server-key.pem":  key,
				},
				Type: "Opaque",
			}
			s, err := coreV1Interface.Secrets(ns).Create(ctx, secret, metav1.CreateOptions{})
			if err != nil {
				log.Infof("Failed to create %s secret in %s namespace.", sn, ns)
				return nil, err
			}
			return s, nil
		}
		log.Infof("error accessing certificate secret %s: %v", sn, err)
		return nil, err
	}
	log.Infof("Secret %s exists.", s.Name)
	return s, nil
}

// Update Cluster intercepter CaBundle.
func createUpdateIntercepterCaBundle(ctx context.Context, ciName string, ns string, caCert []byte, c *rest.Config, log *zap.SugaredLogger) (*v1alpha1.ClusterInterceptor, error) {
	tc, err := triggersclientset.NewForConfig(c)
	if err != nil {
		return nil, err
	}
	ci, err := clusterinterceptorsinformer.Get(ctx).Lister().Get(ciName)
	if err != nil {
		all, err := clusterinterceptorsinformer.Get(ctx).Lister().List(labels.Everything())
		fmt.Println(all, err)
		log.Infof("Server failed to get clusterinterceptor by name, probably clusterinterceptor wasn't created yet %v. Creating...", zap.Error(err))
		var port int32 = 8443
		ci := &v1alpha1.ClusterInterceptor{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ClusterInterceptor",
				APIVersion: "triggers.tekton.dev/v1alpha1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name: ciName,
				Labels: map[string]string{
					"server/type": "https",
				},
			},
			Spec: v1alpha1.ClusterInterceptorSpec{ClientConfig: v1alpha1.ClientConfig{
				CaBundle: caCert,
				Service: &v1alpha1.ServiceReference{
					Name:      ciName,
					Namespace: ns,
					Path:      "/",
					Port:      &port,
				},
			}},
		}
		if ci, err = tc.TriggersV1alpha1().ClusterInterceptors().Create(ctx, ci, metav1.CreateOptions{}); err != nil {
			log.Fatalf("Server failed to create clusterinterceptor with caBundle", zap.Error(err))
		}
		return ci, nil
	} else if !bytes.Equal(ci.Spec.ClientConfig.CaBundle, caCert) {
		// Update cert if the clusterinterceptor exists and caBundle is different from caCert.
		ci.Spec.ClientConfig.CaBundle = caCert
		if _, err = tc.TriggersV1alpha1().ClusterInterceptors().Update(ctx, ci, metav1.UpdateOptions{FieldManager: "custom-interceptor"}); err != nil {
			log.Fatalf("Server failed to update clusterinterceptor caBundle", zap.Error(err))
		}
		log.Infof("CaBundle was updated for %v", ciName)
		return ci, nil
	}
	return nil, nil
}

func PrepareTLS(ctx context.Context, logger *zap.SugaredLogger, c *rest.Config, n, ns string) tls.Certificate {
	secret, err := getCreateCertsSecret(ctx, kubeclient.Get(ctx).CoreV1(), logger, n, ns)
	if err != nil {
		logger.Fatalw("Failed to get or create k8s secret with certificates", zap.Error(err))
	}
	_, err = createUpdateIntercepterCaBundle(ctx, n, ns, secret.Data["ca-cert.pem"], c, logger)
	if err != nil {
		logger.Fatalw("Failed to create or update clusterinterceptor", zap.Error(err))
	}
	certs, err := tls.X509KeyPair(secret.Data["server-cert.pem"], secret.Data["server-key.pem"])
	if err != nil {
		logger.Fatalw("Failed to create X509KeyPair from k8s secret certs", zap.Error(err))
	}
	return certs
}
