package logproxy

import (
	"github.com/go-chi/chi"
	"go.uber.org/zap"
	"knative.dev/pkg/logging"
	"net/http"
)

const DefaultPattern = "/logs/{namespace}/{pod}/{container}"

func NewHandler(
	service Service,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.FromContext(ctx)
		rctx := chi.RouteContext(ctx)
		namespace := rctx.URLParam("namespace")
		pod := rctx.URLParam("pod")
		container := rctx.URLParam("container")
		buf, err := service.Fetch(ctx, namespace, pod, container)
		if err != nil {
			logger.Errorw("Handler failed to fetch logs", zap.Error(err))
			switch err.(type) {
			case *BucketNotExistError, *ObjectNotExistError:
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			default:
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
			return
		}
		_, err = w.Write(buf)
		if err != nil {
			logger.Errorw("Handler failed to write logs", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}
