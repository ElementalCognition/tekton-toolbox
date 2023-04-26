package main

import (
	"context"
	"crypto/tls"
	"flag"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/ElementalCognition/tekton-toolbox/internal/chimiddleware"
	"github.com/ElementalCognition/tekton-toolbox/internal/clusterinterceptorupdater"
	"github.com/ElementalCognition/tekton-toolbox/internal/knativeinjection"
	"github.com/ElementalCognition/tekton-toolbox/internal/serversignals"
	"github.com/ElementalCognition/tekton-toolbox/internal/viperconfig"
	"github.com/ElementalCognition/tekton-toolbox/pkg/cloudeventsync"
	"github.com/ElementalCognition/tekton-toolbox/pkg/githubstatussync"
	"github.com/ElementalCognition/tekton-toolbox/pkg/githubtransport"
	"github.com/ElementalCognition/tekton-toolbox/pkg/triggers"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/google/go-github/v43/github"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/dynamic/dynamicinformer"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
)

type config struct {
	Addr                 string
	GithubAppID          int64  `mapstructure:"github-app-id"`
	GithubInstallationID int64  `mapstructure:"github-installation-id"`
	GithubAppKey         string `mapstructure:"github-app-key"`
}

const (
	component        = "github-status-sync"
	readTimeout      = 5 * time.Second
	writeTimeout     = 20 * time.Second
	idleTimeout      = 60 * time.Second
	forceStopTimeout = 1 * time.Minute
)

func newGithubClient(cfg *config) (*github.Client, error) {
	githubTransport, err := githubtransport.NewTransport(cfg.GithubAppID, cfg.GithubInstallationID, cfg.GithubAppKey)
	if err != nil {
		return nil, err
	}
	return github.NewClient(&http.Client{Transport: githubTransport}), nil
}

func newMux(
	service cloudeventsync.Service,
	logger *zap.SugaredLogger,
) *chi.Mux {
	mux := chi.NewRouter()
	mux.Group(func(r chi.Router) {
		chimiddleware.WithHeartbeat(r)
	})
	mux.Group(func(r chi.Router) {
		r.Use(middleware.RequestID)
		r.Use(chimiddleware.WithRequestID)
		r.Use(chimiddleware.RequestLogger(logger))
		r.Use(middleware.Recoverer)
		r.Post("/", triggers.NewHandler(
			cloudeventsync.NewInterceptor(
				service,
			),
		))
	})
	return mux
}

func getIntercepterName() string {
	// Keep k8s service name and clusterintercepter name the same.
	if ci, ok := os.LookupEnv("INTERCEPTER_NAME"); ok {
		return ci
	}
	return "github-status-sync"
}

func newClusterInterceptorInformer(kubeCfg *rest.Config, logger *zap.SugaredLogger) (dynamicinformer.DynamicSharedInformerFactory, cache.SharedIndexInformer) {
	// Create a dynamic client
	dynamicClient, err := dynamic.NewForConfig(kubeCfg)
	if err != nil {
		logger.Fatalw("Server failed to create dynamic client", zap.Error(err))
	}

	// Define the GVR for ClusterInterceptor
	clusterInterceptorGVR := schema.GroupVersionResource{
		Group:    "triggers.tekton.dev",
		Version:  "v1alpha1",
		Resource: "clusterinterceptors",
	}

	// Create a dynamic informer factory
	dynamicInformerFactory := dynamicinformer.NewDynamicSharedInformerFactory(dynamicClient, 0)

	// Get the ClusterInterceptor dynamic informer
	clusterInterceptorInformer := dynamicInformerFactory.ForResource(clusterInterceptorGVR).Informer()

	return dynamicInformerFactory, clusterInterceptorInformer
}

func init() {
	flag.String("config", "", "The path to the config file.")
	flag.String("addr", "0.0.0.0:8443", "The address and port.")
	flag.Int64("github-app-id", 0, "GitHub App ID.")
	flag.Int64("github-installation-id", 0, "GitHub Installation ID.")
	flag.String("github-app-key", "", "GitHub App key.")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
}

func main() {
	ctx := signals.NewContext()
	// Initialize logging and config
	logger := knativeinjection.SetupLoggerOrDie(ctx, component)
	ctx = logging.WithLogger(ctx, logger)
	viperCfg, err := viperconfig.NewConfig(component, pflag.CommandLine)
	if err != nil {
		logger.Fatalw("Server failed to initialize config", zap.Error(err))
	}
	var cfg config
	err = viperconfig.LoadConfig(viperCfg, &cfg)
	if err != nil {
		logger.Fatalw("Server failed to load config", zap.Error(err))
	}
	// Set up Kubernetes client and informers
	kubeCfg := injection.ParseAndGetRESTConfigOrDie()
	if err != nil {
		logger.Fatalw("Server failed to create Kubernetes client", zap.Error(err))
	}
	// Create the ClusterInterceptor dynamic informer and informer factory
	dynamicInformerFactory, clusterInterceptorInformer := newClusterInterceptorInformer(kubeCfg, logger)
	dynamicInformerFactory.Start(ctx.Done())
	// Wait for the cache to sync
	if !cache.WaitForCacheSync(ctx.Done(), clusterInterceptorInformer.HasSynced) {
		logger.Fatalw("Failed to sync cache for informer")
	}
	githubClient, err := newGithubClient(&cfg)
	if err != nil {
		logger.Fatalw("Server failed to create GitHub client", zap.Error(err))
	}
	svc := githubstatussync.NewService(githubClient)
	// Set up TLS and interceptor name
	interceptorName := getIntercepterName()
	ns := clusterinterceptorupdater.GetNamespace()
	certs := clusterinterceptorupdater.PrepareTLS(ctx, logger, kubeCfg, interceptorName, ns)
	// Set up server and mux
	mux := newMux(svc, logger)
	srv := &http.Server{
		Addr:         cfg.Addr,
		TLSConfig:    &tls.Config{Certificates: []tls.Certificate{certs}},
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
		Handler:      mux,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}
	// Start server
	s := serversignals.Server{
		Server:           srv,
		Logger:           logger,
		ForceStopTimeout: forceStopTimeout,
	}
	if err := s.StartAndWaitSignalsThenShutdown(context.Background()); err != http.ErrServerClosed {
		logger.Fatalw("Server failed to shutdown", zap.Error(err))
	}
}
