package main

import (
	"context"
	"flag"
	"net"
	"net/http"
	"runtime"
	"time"

	"cloud.google.com/go/storage"
	"github.com/ElementalCognition/tekton-toolbox/internal/chimiddleware"
	"github.com/ElementalCognition/tekton-toolbox/internal/knativeinjection"
	"github.com/ElementalCognition/tekton-toolbox/internal/serversignals"
	"github.com/ElementalCognition/tekton-toolbox/internal/viperconfig"
	"github.com/ElementalCognition/tekton-toolbox/pkg/gcslogproxy"
	"github.com/ElementalCognition/tekton-toolbox/pkg/logproxy"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/spf13/pflag"
	"go.uber.org/zap"
	"google.golang.org/api/option"
	"gopkg.in/go-playground/pool.v3"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/logging"
)

type config struct {
	Bucket          string
	Addr            string
	Workers         uint
	CredentialsFile string
}

const (
	component        = "gcs-log-proxy"
	readTimeout      = 5 * time.Second
	writeTimeout     = 20 * time.Second
	idleTimeout      = 60 * time.Second
	forceStopTimeout = 1 * time.Minute
)

func newStorageClient(credentialsFile string) (*storage.Client, error) {
	var client *storage.Client
	var err error
	if len(credentialsFile) != 0 {
		client, err = storage.NewClient(context.Background(), option.WithCredentialsFile(credentialsFile))
	} else {
		client, err = storage.NewClient(context.Background())
	}
	return client, err
}

func newMux(
	service logproxy.Service,
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
		r.Get(logproxy.DefaultPattern, logproxy.NewHandler(service))
	})
	return mux
}

func init() {
	flag.String("config", "", "The path to the config file.")
	flag.String("bucket", "", "The logs bucket.")
	flag.String("addr", "0.0.0.0:80", "The address and port.")
	flag.Int("workers", runtime.NumCPU(), "The number of workers to fetch logs.")
	flag.String("credentials", "", "The path to the keyfile. If not present, client will use your default application credentials.")
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
	storageClient, err := newStorageClient(cfg.CredentialsFile)
	if err != nil {
		logger.Fatalw("Server failed to create GCS client", zap.Error(err))
	}
	p := pool.NewLimited(cfg.Workers)
	svc := gcslogproxy.NewService(cfg.Bucket, storageClient, p)
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
	srv.RegisterOnShutdown(func() {
		if err := storageClient.Close(); err != nil {
			logger.Errorw("Server failed to close GCS client", zap.Error(err))
		}
	})
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
