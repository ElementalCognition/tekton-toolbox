package serversignals

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type Server struct {
	*http.Server
	Logger           *zap.SugaredLogger
	ForceStopTimeout time.Duration
}

func (s *Server) Shutdown(ctx context.Context) error {
	g, gctx := errgroup.WithContext(ctx)
	timer := time.NewTimer(s.ForceStopTimeout)
	g.Go(func() error {
		select {
		case <-timer.C:
			s.Logger.Infow("Server force closed")
			return s.Server.Close()
		case <-gctx.Done():
			return gctx.Err()
		}
	})
	g.Go(func() error {
		defer timer.Stop()
		s.Logger.Infow("Server closed")
		return s.Server.Shutdown(ctx)
	})
	return g.Wait()
}

func (s *Server) WaitSignalsThenShutdown(ctx context.Context) error {
	g, gctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGTERM, syscall.SIGINT)
		select {
		case sig := <-c:
			s.Logger.Infow("Server received OS signal", zap.String("sig", sig.String()))
			return s.Shutdown(ctx)
		case <-gctx.Done():
			return gctx.Err()
		}
	})
	return g.Wait()
}

func (s *Server) ListenAndServe() error {
	s.Logger.Infow("Server started")
	s.Logger.Infow("Server listen and serve", zap.String("addr", s.Addr))
	if s.TLSConfig != nil {
		return s.Server.ListenAndServeTLS("", "")
	}
	return s.Server.ListenAndServe()
}

func (s *Server) StartAndWaitSignalsThenShutdown(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		return s.WaitSignalsThenShutdown(ctx)
	})
	g.Go(func() error {
		return s.ListenAndServe()
	})
	return g.Wait()
}
