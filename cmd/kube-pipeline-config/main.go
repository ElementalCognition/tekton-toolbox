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
	"github.com/ElementalCognition/tekton-toolbox/pkg/kubepipelineconfig"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineresolver"
	"github.com/ElementalCognition/tekton-toolbox/pkg/triggers"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
)

type config struct {
	Addr string
}

const (
	component        = "kube-pipeline-config"
	readTimeout      = 5 * time.Second
	writeTimeout     = 20 * time.Second
	idleTimeout      = 60 * time.Second
	forceStopTimeout = 1 * time.Second
)

func newMux(
	service kubepipelineconfig.Service,
	resolver pipelineresolver.Resolver,
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
			kubepipelineconfig.NewInterceptor(
				service,
				resolver,
			),
		))
	})
	return mux
}

func init() {
	flag.String("config", "", "The path to the config file.")
	flag.String("addr", "0.0.0.0:8443", "The address and port.")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
}

func main() {
	ctx := signals.NewContext()
	kubeCfg := injection.ParseAndGetRESTConfigOrDie()
	ctx, startInformer := injection.EnableInjectionOrDie(ctx, kubeCfg)
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
	resolver, err := pipelineresolver.NewCelResolver()
	if err != nil {
		logger.Fatalw("Server failed to create CEL resolver", zap.Error(err))
	}
	kubeClient, err := kubernetes.NewForConfig(kubeCfg)
	if err != nil {
		logger.Fatalw("Server failed to create Kubernetes client", zap.Error(err))
	}
	svc := kubepipelineconfig.NewService(kubeClient)
	err = svc.Start(context.Background())
	if err != nil {
		logger.Fatalw("Server failed to start Kubernetes service", zap.Error(err))
	}

	startInformer()
	intercepterName, ok := os.LookupEnv("INTERCEPTER_NAME")
	if !ok {
		intercepterName = "kube-pipeline-config"
	}
	crt, caCert, err := clusterinterceptorupdater.GenerateCertificates(ctx, intercepterName)
	if err != nil {
		logger.Fatalw("Failed to generate certificates", zap.Error(err))
	}
	err = clusterinterceptorupdater.UpdateIntercepterCaBundle(ctx, intercepterName, caCert, kubeCfg, logger)
	if err != nil {
		logger.Fatalw("Failed to update cluster intercepter caBundle", zap.Error(err))
	}

	mux := newMux(svc, resolver, logger)
	srv := &http.Server{
		Addr:         cfg.Addr,
		TLSConfig:    &tls.Config{Certificates: []tls.Certificate{*crt}},
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
		Handler:      mux,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}
	srv.RegisterOnShutdown(func() {
		err := svc.Close()
		if err != nil {
			logger.Fatalw("Server failed to close Kubernetes service", zap.Error(err))
		}
	})
	s := serversignals.Server{
		Server:           srv,
		Logger:           logger,
		ForceStopTimeout: forceStopTimeout,
	}
	if err := s.StartAndWaitSignalsThenShutdown(context.Background()); err != http.ErrServerClosed {
		logger.Fatalw("Server failed to shutdown", zap.Error(err))
	}
}
