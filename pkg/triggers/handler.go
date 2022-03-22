package triggers

import (
	"encoding/json"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"knative.dev/pkg/logging"
	"net/http"
)

func NewHandler(
	interceptor v1beta1.InterceptorInterface,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		logger := logging.FromContext(ctx)
		var req v1beta1.InterceptorRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			logger.Errorw("Handler failed to parse request")
			http.Error(w, http.StatusText(http.StatusBadRequest), http.StatusBadRequest)
			return
		}
		logger.Infow("Handler started request",
			zap.Any("context", req.Context),
			zap.Any("extensions", req.Extensions),
			zap.Any("header", req.Header),
			zap.Any("params", req.InterceptorParams),
		)
		res := interceptor.Process(ctx, &req)
		rlogger := logger.With(
			zap.Bool("continue", res.Continue),
			zap.Any("extensions", res.Extensions),
			zap.String("status", res.Status.Err().Error()),
		)
		if res.Status.Code != codes.OK {
			rlogger.Errorw("Handler finished request")
		} else {
			rlogger.Infow("Handler finished request")
		}
		w.Header().Add("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(res); err != nil {
			logger.Errorw("Handler failed to write response", zap.Error(err))
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}
