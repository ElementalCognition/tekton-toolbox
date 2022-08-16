package clusterinterceptorupdater

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/webhook/certificates/resources"

	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	clusterinterceptorsinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/clusterinterceptor"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getNamespace() string {
	if ns := os.Getenv("SVC_NAMESPACE"); ns != "" {
		return ns
	}
	return "tekton-pipelines"
}

func GenerateCertificates(ctx context.Context, inName string) (*tls.Certificate, []byte, error) {
	expiration := time.Now().AddDate(10, 0, 0)
	ns := getNamespace()
	fmt.Printf("Generate certificates for svc %s in %s namespace.\n", inName, ns)
	key, cert, caCert, err := resources.CreateCerts(ctx, inName, ns, expiration)
	if err != nil {
		return &tls.Certificate{}, nil, err
	}
	crt, err := tls.X509KeyPair(cert, key)
	if err != nil {
		return &tls.Certificate{}, nil, err
	}
	caBundle := append(caCert, cert...)
	return &crt, caBundle, nil
}

// Update Cluster intercepter CaBundle
func UpdateIntercepterCaBundle(ctx context.Context, inName string, caCert []byte, c *rest.Config, log *zap.SugaredLogger) error {
	tc, err := triggersclientset.NewForConfig(c)
	if err != nil {
		return err
	}
	intercepter, err := clusterinterceptorsinformer.Get(ctx).Lister().Get(inName)
	if err != nil {
		log.Info("Server failed to get clusterinterceptor by name, probabley clusterinterceptor wasn't created yet ", zap.Error(err))
	} else {
		// update cert on creation if the clusterinterceptor exists
		intercepter.Spec.ClientConfig.CaBundle = caCert
		if _, err = tc.TriggersV1alpha1().ClusterInterceptors().Update(ctx, intercepter, metav1.UpdateOptions{TypeMeta: metav1.TypeMeta{Kind: "ClusterInterceptor"},
			FieldManager: "pipeline-config-trigger"}); err != nil {
			log.Info("Server failed to update clusterinterceptor cabundle", zap.Error(err))
		}
	}
	wi, _ := tc.TriggersV1alpha1().ClusterInterceptors().Watch(ctx, metav1.ListOptions{TypeMeta: metav1.TypeMeta{Kind: "ClusterInterceptor"},
		FieldSelector: fmt.Sprintf("metadata.name=%s", inName)})
	if err != nil {
		log.Info("Server failed to watch ClusterInterceptors", zap.Error(err))
	}
	log.Info("Watching clusterInterceptor: ", inName)
	go func(wi watch.Interface) {
		for {
			switch event := <-wi.ResultChan(); event.Type {
			case "MODIFIED": //"ADDED",
				log.Info("Clusterinterceptor was added ", inName)
				intercepter, err := clusterinterceptorsinformer.Get(ctx).Lister().Get(inName)
				if err != nil {
					log.Info("Server failed to get clusterinterceptor by name, probabley clusterinterceptor wasn't created yet", zap.Error(err))
					break
				}
				if res := bytes.Compare(intercepter.Spec.ClientConfig.CaBundle, caCert); res == 0 {
					log.Info("CaBundle is up to date, there is nothing to change.")
					break
				}
				intercepter.Spec.ClientConfig.CaBundle = caCert
				if err != nil {
					log.Info("Server failed to get clusterinterceptor by name, probabley clusterinterceptor wasn't created yet", zap.Error(err))
				}
				if _, err = tc.TriggersV1alpha1().ClusterInterceptors().Update(ctx, intercepter, metav1.UpdateOptions{TypeMeta: metav1.TypeMeta{Kind: "ClusterInterceptor"},
					FieldManager: "pipeline-config-trigger"}); err != nil {
					log.Info("Server failed to update clusterinterceptor cabundle", zap.Error(err))
				}
			case "DELETED":
				log.Info("ClusterInterceptor was deleted: ", inName)
			case "ERROR":
				log.Info("ClusterInterceptor event ERROR: ", event.Object)
			}
		}
	}(wi)
	go func(wi watch.Interface, ctx context.Context) {
		<-ctx.Done()
		wi.Stop()
	}(wi, ctx)
	return nil
}
