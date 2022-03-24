package chimiddleware

import (
	"context"
	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
	"knative.dev/pkg/logging"
	"net/http"
)

func RequestID(ctx context.Context) zap.Field {
	if reqID := middleware.GetReqID(ctx); reqID != "" {
		return zap.String("requestId", reqID)
	}
	return zap.Skip()
}

func WithRequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.FromContext(ctx).With(RequestID(ctx))
		ctx = logging.WithLogger(ctx, logger)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
