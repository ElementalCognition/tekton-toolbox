package main

import (
	"context"
	"flag"
	"github.com/ElementalCognition/tekton-toolbox/internal/chimiddleware"
	"github.com/ElementalCognition/tekton-toolbox/internal/knativeinjection"
	"github.com/ElementalCognition/tekton-toolbox/internal/serversignals"
	"github.com/ElementalCognition/tekton-toolbox/internal/viperconfig"
	"github.com/ElementalCognition/tekton-toolbox/pkg/cloudeventsync"
	"github.com/ElementalCognition/tekton-toolbox/pkg/githubstatussync"
	"github.com/ElementalCognition/tekton-toolbox/pkg/githubtransport"
	"github.com/ElementalCognition/tekton-toolbox/pkg/triggers"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/google/go-github/v39/github"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
	"net"
	"net/http"
	"time"
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

func init() {
	flag.String("config", "", "The path to the config file.")
	flag.String("addr", "0.0.0.0:80", "The address and port.")
	flag.Int64("github-app-id", 0, "GitHub App ID.")
	flag.Int64("github-installation-id", 0, "GitHub Installation ID.")
	flag.String("github-app-key", "", "GitHub App key.")
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
	githubClient, err := newGithubClient(&cfg)
	if err != nil {
		logger.Fatalw("Server failed to create GitHub client", zap.Error(err))
	}
	svc := githubstatussync.NewService(githubClient)
	mux := newMux(svc, logger)
	srv := &http.Server{
		Addr:         cfg.Addr,
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
