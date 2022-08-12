package main

import (
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/ElementalCognition/tekton-toolbox/internal/chimiddleware"
	"github.com/ElementalCognition/tekton-toolbox/internal/knativeinjection"
	"github.com/ElementalCognition/tekton-toolbox/internal/serversignals"
	"github.com/ElementalCognition/tekton-toolbox/internal/viperconfig"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineconfigtrigger"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineresolver"
	"github.com/ElementalCognition/tekton-toolbox/pkg/triggers"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/spf13/pflag"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"github.com/tektoncd/triggers/pkg/interceptors/server"
	"go.uber.org/zap"
	"gopkg.in/go-playground/pool.v3"
	kubeclientset "k8s.io/client-go/kubernetes"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
)

type config struct {
	Addr    string
	Workers uint
}

const (
	component        = "pipeline-config-trigger"
	readTimeout      = 5 * time.Second
	writeTimeout     = 20 * time.Second
	idleTimeout      = 60 * time.Second
	forceStopTimeout = 1 * time.Second
)

func newMux(
	service pipelineconfigtrigger.Service,
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
			pipelineconfigtrigger.NewInterceptor(
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
	flag.Int("workers", runtime.NumCPU(), "The number of workers to trigger pipelines.")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
}

func main() {
	kubeCfg := injection.ParseAndGetRESTConfigOrDie()
	ctx := knativeinjection.EnableInjectionOrDie(kubeCfg)
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
	tektonClient, err := versioned.NewForConfig(kubeCfg)
	if err != nil {
		logger.Fatalw("Server failed to create Tekton client", zap.Error(err))
	}
	resolver, err := pipelineresolver.NewCelResolver()
	if err != nil {
		logger.Fatalw("Server failed to create CEL resolver", zap.Error(err))
	}

	kubeClient, err := kubeclientset.NewForConfig(kubeCfg)
	if err != nil {
		logger.Errorf("failed to create new Clientset for the given config: %v", err)
	}
	keyFile, certFile, _, err := server.CreateCerts(ctx, kubeClient.CoreV1(), logger)
	if err != nil {
		fmt.Println(certFile, keyFile)
		logger.Fatalw("Server failed to create Certs", zap.Error(err))
	}
	crt, err := tls.X509KeyPair([]byte(certFile), []byte(keyFile))
	if err != nil {
		logger.Fatalw("Server failed to create X509KeyPair", zap.Error(err))
	}

	p := pool.NewLimited(cfg.Workers)
	svc := pipelineconfigtrigger.NewService(tektonClient, p)
	mux := newMux(svc, resolver, logger)
	srv := &http.Server{
		Addr:         cfg.Addr,
		TLSConfig:    &tls.Config{Certificates: []tls.Certificate{crt}},
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
		Handler:      mux,
		BaseContext: func(_ net.Listener) context.Context {
			return ctx
		},
	}
	srv.RegisterOnShutdown(func() {
		p.Close()
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
