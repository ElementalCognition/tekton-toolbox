package main

import (
	"context"
	"crypto/tls"
	"flag"
	"net"
	"net/http"
	"os"
	"runtime"
	"time"

	"github.com/ElementalCognition/tekton-toolbox/internal/chimiddleware"
	"github.com/ElementalCognition/tekton-toolbox/internal/clusterinterceptorupdater"
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
	"go.uber.org/zap"
	"gopkg.in/go-playground/pool.v3"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
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

func getIntercepterName() string {
	// Keep k8s service name and clusterintercepter name the same.
	if ci, ok := os.LookupEnv("INTERCEPTER_NAME"); ok {
		return ci
	}
	return "pipeline-config-trigger"
}

func prepareTLS(ctx context.Context, logger *zap.SugaredLogger, rc *rest.Config, n, ns string) tls.Certificate {
	secret, err := clusterinterceptorupdater.GetCreateCertsSecret(ctx, kubeclient.Get(ctx).CoreV1(), logger, n, ns)
	if err != nil {
		logger.Fatalw("Failed to get or create k8s secret with certificates", zap.Error(err))
	}
	err = clusterinterceptorupdater.CreateUpdateIntercepterCaBundle(ctx, n, ns, secret.Data["ca-cert.pem"], rc, logger)
	if err != nil {
		logger.Fatalw("Failed to create or update clusterinterceptor", zap.Error(err))
	}
	certs, err := tls.X509KeyPair(secret.Data["server-cert.pem"], secret.Data["server-key.pem"])
	if err != nil {
		logger.Fatalw("Failed to create X509KeyPair from k8s secret certs", zap.Error(err))
	}
	return certs
}

func init() {
	flag.String("config", "", "The path to the config file.")
	flag.String("addr", "0.0.0.0:8443", "The address and port.")
	flag.Int("workers", runtime.NumCPU(), "The number of workers to trigger pipelines.")
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
	tektonClient, err := versioned.NewForConfig(kubeCfg)
	if err != nil {
		logger.Fatalw("Server failed to create Tekton client", zap.Error(err))
	}
	resolver, err := pipelineresolver.NewCelResolver()
	if err != nil {
		logger.Fatalw("Server failed to create CEL resolver", zap.Error(err))
	}
	startInformer()
	intercepterName := getIntercepterName()
	ns := clusterinterceptorupdater.GetNamespace()
	certs := prepareTLS(ctx, logger, kubeCfg, intercepterName, ns)
	p := pool.NewLimited(cfg.Workers)
	svc := pipelineconfigtrigger.NewService(tektonClient, p)
	mux := newMux(svc, resolver, logger)
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
