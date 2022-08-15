package clusterinterceptorupdater

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/system"
	"knative.dev/pkg/webhook/certificates/resources"

	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	clusterinterceptorsinformer "github.com/tektoncd/triggers/pkg/client/injection/informers/triggers/v1alpha1/clusterinterceptor"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func GenerateCertificates(ctx context.Context, inName string) (*tls.Certificate, []byte, error) {
	expiration := time.Now().AddDate(10, 0, 0)
	key, cert, caCert, err := resources.CreateCerts(ctx, inName, system.Namespace(), expiration)
	if err != nil {
		return &tls.Certificate{}, nil, err
	}
	crt, err := tls.X509KeyPair([]byte(cert), []byte(key))
	if err != nil {
		return &tls.Certificate{}, nil, err
	}
	return &crt, caCert, nil
}

// Update Cluster intercepter CaBundle, in intercepter name
func UpdateIntercepterCaBundle(ctx context.Context, inName string, caCert []byte, c *rest.Config) error {
	tc, err := triggersclientset.NewForConfig(c)
	if err != nil {
		return err
	}
	intercepter, err := clusterinterceptorsinformer.Get(ctx).Lister().Get(inName)
	if err != nil {
		fmt.Println("Server failed to get clusterinterceptor by name, probabley clusterinterceptor wasn't created yet ", err)
	} else {
		// update cert on creation if the clusterinterceptor exists
		intercepter.Spec.ClientConfig.CaBundle = caCert
		if _, err = tc.TriggersV1alpha1().ClusterInterceptors().Update(ctx, intercepter, metav1.UpdateOptions{TypeMeta: metav1.TypeMeta{Kind: "ClusterInterceptor"},
			FieldManager: "pipeline-config-trigger"}); err != nil {
			fmt.Println("Server failed to update clusterinterceptor cabundle", err)
		}
	}
	wi, _ := tc.TriggersV1alpha1().ClusterInterceptors().Watch(ctx, metav1.ListOptions{TypeMeta: metav1.TypeMeta{Kind: "ClusterInterceptor"},
		FieldSelector: fmt.Sprintf("metadata.name=%s", inName)})
	if err != nil {
		fmt.Println("Server failed to watch ClusterInterceptors", err)
	}
	fmt.Println("Watching clusterInterceptor: ", inName)
	go func(wi watch.Interface) {
		for {
			switch event := <-wi.ResultChan(); event.Type {
			case "MODIFIED": //"ADDED",
				fmt.Println("Clusterinterceptor was added ", inName)
				intercepter, err := clusterinterceptorsinformer.Get(ctx).Lister().Get(inName)
				if err != nil {
					fmt.Println("Server failed to get clusterinterceptor by name, probabley clusterinterceptor wasn't created yet", err)
					break
				}
				if res := bytes.Compare(intercepter.Spec.ClientConfig.CaBundle, caCert); res == 0 {
					fmt.Println("CaBundle is up to date, there is nothing to change.")
					break
				}
				intercepter.Spec.ClientConfig.CaBundle = caCert
				if err != nil {
					fmt.Println("Server failed to get clusterinterceptor by name, probabley clusterinterceptor wasn't created yet", zap.Error(err))
				}
				if _, err = tc.TriggersV1alpha1().ClusterInterceptors().Update(ctx, intercepter, metav1.UpdateOptions{TypeMeta: metav1.TypeMeta{Kind: "ClusterInterceptor"},
					FieldManager: "pipeline-config-trigger"}); err != nil {
					fmt.Println("Server failed to update clusterinterceptor cabundle", err)
				}
			case "DELETED":
				fmt.Println("ClusterInterceptor was deleted: ", inName)
			case "ERROR":
				fmt.Println("ClusterInterceptor event ERROR: ", event.Object)
			}
		}
	}(wi)
	go func(wi watch.Interface, ctx context.Context) {
		<-ctx.Done()
		wi.Stop()
	}(wi, ctx)
	return nil
}
