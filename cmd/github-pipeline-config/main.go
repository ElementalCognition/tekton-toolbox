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
	"github.com/ElementalCognition/tekton-toolbox/pkg/githubpipelineconfig"
	"github.com/ElementalCognition/tekton-toolbox/pkg/githubtransport"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineresolver"
	"github.com/ElementalCognition/tekton-toolbox/pkg/triggers"
	"github.com/ElementalCognition/tekton-toolbox/pkg/vcspipelineconfig"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/google/go-github/v48/github"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
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
	component        = "github-pipeline-config"
	readTimeout      = 5 * time.Second
	writeTimeout     = 20 * time.Second
	idleTimeout      = 60 * time.Second
	forceStopTimeout = 1 * time.Minute
)

func init() {
	flag.String("config", "", "The path to the config file.")
	flag.String("addr", "0.0.0.0:8443", "The address and port.")
	flag.Int64("github-app-id", 0, "GitHub App ID.")
	flag.Int64("github-installation-id", 0, "GitHub Installation ID.")
	flag.String("github-app-key", "", "GitHub App key.")
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	pflag.Parse()
}

func newGithubClient(cfg *config) (*github.Client, error) {
	githubTransport, err := githubtransport.NewTransport(cfg.GithubAppID, cfg.GithubInstallationID, cfg.GithubAppKey)
	if err != nil {
		return nil, err
	}
	return github.NewClient(&http.Client{Transport: githubTransport}), nil
}

func newMux(
	service vcspipelineconfig.Service,
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
			vcspipelineconfig.NewInterceptor(
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
	return "github-pipeline-config"
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
	githubClient, err := newGithubClient(&cfg)
	if err != nil {
		logger.Fatalw("Server failed to create GitHub client", zap.Error(err))
	}
	resolver, err := pipelineresolver.NewCelResolver()
	if err != nil {
		logger.Fatalw("Server failed to create CEL resolver", zap.Error(err))
	}
	svc := githubpipelineconfig.NewService(githubClient)
	startInformer()
	intercepterName := getIntercepterName()
	ns := clusterinterceptorupdater.GetNamespace()
	certs := clusterinterceptorupdater.PrepareTLS(ctx, logger, kubeCfg, intercepterName, ns)
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
	s := serversignals.Server{
		Server:           srv,
		Logger:           logger,
		ForceStopTimeout: forceStopTimeout,
	}
	if err := s.StartAndWaitSignalsThenShutdown(context.Background()); err != http.ErrServerClosed {
		logger.Fatalw("Server failed to shutdown", zap.Error(err))
	}
}
