package kubepipelineconfig

import (
	"context"
	"fmt"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineconfig"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	watchext "k8s.io/client-go/tools/watch"
	"knative.dev/pkg/logging"
	"strings"
	"sync"
)

type Service interface {
	Start(ctx context.Context) error
	Get(ctx context.Context, namespace, name string) (*pipelineconfig.Config, error)
	Close() error
}

type service struct {
	kubeClient kubernetes.Interface
	mutex      sync.Mutex
	cache      sync.Map
	watcher    *watchext.RetryWatcher
}

var _ Service = (*service)(nil)

func keyFor(namespace string, name string) string {
	return fmt.Sprintf("%s.%s", name, namespace)
}

func (s *service) Start(ctx context.Context) error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.watcher == nil {
		labelSelector := labels.SelectorFromSet(labels.Set{
			"name": fmt.Sprintf("%s-cm", pipelineconfig.ConfigKey),
		})
		rw, err := watchext.NewRetryWatcher("latest", &cache.ListWatch{
			WatchFunc: func(options metav1.ListOptions) (watch.Interface, error) {
				options.LabelSelector = labelSelector.String()
				if options.ResourceVersion == "latest" {
					options.ResourceVersion = ""
				}
				return s.kubeClient.CoreV1().ConfigMaps(metav1.NamespaceAll).Watch(ctx, options)
			},
		})
		if err != nil {
			return nil
		}
		s.watcher = rw
		go s.watch(ctx)
	}
	return nil
}

func (s *service) watch(ctx context.Context) {
	for {
		e := <-s.watcher.ResultChan()
		if e.Type == watch.Error && e.Object == nil {
			return
		}
		cm := e.Object.(*v1.ConfigMap)
		key := keyFor(cm.Namespace, cm.Name)
		logger := logging.FromContext(ctx)
		logger.Infow("Service received config update",
			zap.String("namespace", cm.Namespace),
			zap.String("name", cm.Name),
			zap.String("event", strings.ToLower(string(e.Type))),
		)
		switch e.Type {
		case watch.Added, watch.Modified:
			cfg := &pipelineconfig.Config{}
			err := cfg.UnmarshalConfigMapYAML(cm)
			if err == nil {
				s.cache.Store(key, cfg)
			}
		case watch.Deleted, watch.Error:
			s.cache.Delete(key)
		}
	}
}

func (s *service) Get(ctx context.Context, namespace string, name string) (*pipelineconfig.Config, error) {
	logger := logging.FromContext(ctx)
	key := keyFor(namespace, name)
	v, ok := s.cache.Load(key)
	if ok {
		logger.Infow("Service fetched config; cached",
			zap.String("namespace", namespace),
			zap.String("name", name),
		)
		return v.(*pipelineconfig.Config), nil
	}
	v, ok = s.cache.Load(key)
	if !ok {
		logger.Infow("Service fetched config; remote",
			zap.String("namespace", namespace),
			zap.String("name", name),
		)
		cm, err := s.kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return nil, err
		}
		cfg := &pipelineconfig.Config{}
		err = cfg.UnmarshalConfigMapYAML(cm)
		if err != nil {
			return nil, err
		}
		s.cache.Store(key, cfg)
	}
	return v.(*pipelineconfig.Config), nil
}

func (s *service) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	if s.watcher != nil {
		s.watcher.Stop()
	}
	return nil
}

func NewService(
	kubeClient kubernetes.Interface,
) Service {
	return &service{
		kubeClient: kubeClient,
	}
}
