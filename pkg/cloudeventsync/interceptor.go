package cloudeventsync

import (
	"context"
	"encoding/json"
	"github.com/tektoncd/pipeline/pkg/reconciler/events/cloudevent"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"knative.dev/pkg/logging"
	"strings"
)

type interceptor struct {
	service Service
}

var _ v1beta1.InterceptorInterface = (*interceptor)(nil)

func (i *interceptor) Process(ctx context.Context, req *v1beta1.InterceptorRequest) *v1beta1.InterceptorResponse {
	logger := logging.FromContext(ctx)
	ceType, ok := req.Header["Ce-Type"]
	if !ok || len(ceType[0]) == 0 {
		logger.Warnw("Interceptor received empty cloud event; skipping")
		return interceptors.Fail(codes.InvalidArgument, "Only cloud event is allowed")
	}
	var ce cloudevent.TektonCloudEventData
	if err := json.NewDecoder(strings.NewReader(req.Body)).Decode(&ce); err != nil {
		logger.Errorw("Interceptor failed to unmarshal cloud event", zap.Error(err))
		return interceptors.Fail(codes.InvalidArgument, "Cloud event is malformed")
	}
	err := i.service.Sync(ctx, ceType[0], &ce)
	if err != nil {
		logger.Errorw("Interceptor failed to sync status", zap.Error(err))
		return interceptors.Fail(codes.Internal, "Unable to sync status")
	}
	return &v1beta1.InterceptorResponse{
		Continue: false,
		Status: v1beta1.Status{
			Code: codes.OK,
		},
	}
}

func NewInterceptor(
	service Service,
) v1beta1.InterceptorInterface {
	return &interceptor{
		service: service,
	}
}
