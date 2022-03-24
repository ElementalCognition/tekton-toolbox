package knativeinjection

import (
	"context"
	"go.uber.org/zap"
	"knative.dev/pkg/injection/sharedmain"
)

func SetupLoggerOrDie(ctx context.Context, component string) *zap.SugaredLogger {
	logger, atomicLevel := sharedmain.SetupLoggerOrDie(ctx, component)
	cmw := sharedmain.SetupConfigMapWatchOrDie(ctx, logger)
	sharedmain.WatchLoggingConfigOrDie(ctx, cmw, logger, atomicLevel, component)
	return logger
}
