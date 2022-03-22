package chimiddleware

import (
	"fmt"
	"github.com/go-chi/chi/middleware"
	"go.uber.org/zap"
	"knative.dev/pkg/logging"
	"net/http"
	"time"
)

var _ middleware.LogFormatter = (*LogFormatter)(nil)
var _ middleware.LogEntry = (*LogEntry)(nil)

type LogEntry struct {
	Logger *zap.SugaredLogger
}

func (e *LogEntry) Write(status, bytes int, _ http.Header, elapsed time.Duration, _ interface{}) {
	logger := e.Logger.With(
		zap.Int("status", status),
		zap.Int("responseSize", bytes),
		zap.Duration("latency", elapsed),
	)
	if status != http.StatusOK {
		logger.Errorw("Server finished request")
	} else {
		logger.Infow("Server finished request")
	}
}

func (e *LogEntry) Panic(v interface{}, stack []byte) {
	e.Logger.Errorw("Server failed with unhandled error",
		zap.String("stack", string(stack)),
		zap.String("panic", fmt.Sprintf("%+v", v)),
	)
}

type LogFormatter struct {
	Logger *zap.SugaredLogger
}

func (f *LogFormatter) NewLogEntry(r *http.Request) middleware.LogEntry {
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	logger := logging.FromContext(r.Context())
	logger.Infow("Server started request",
		zap.String("requestMethod", r.Method),
		zap.String("protocol", r.Proto),
		zap.String("remoteIp", r.RemoteAddr),
		zap.String("userAgent", r.UserAgent()),
		zap.String("requestUrl", fmt.Sprintf("%s://%s%s", scheme, r.Host, r.RequestURI)),
	)
	return &LogEntry{
		Logger: logger,
	}
}

func RequestLogger(logger *zap.SugaredLogger) func(next http.Handler) http.Handler {
	return middleware.RequestLogger(&LogFormatter{
		Logger: logger,
	})
}
