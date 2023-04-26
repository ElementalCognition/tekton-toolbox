package cloudeventsync

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/tektoncd/pipeline/pkg/reconciler/events/cloudevent"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/logging"
)

type interceptor struct {
	service Service
}

var _ v1beta1.InterceptorInterface = (*interceptor)(nil)

type trState struct {
	status string
}

// TaskRun status stor.
var trss = map[types.UID]*trState{}

// Events skipped.
var es int

func checkRunStatusChanged(s string, ce *cloudevent.TektonCloudEventData) bool {
	tr := ce.TaskRun
	if tr == nil {
		return false
	}
	if val, ok := trss[tr.UID]; ok {
		if val.status == s {
			es++
			return false
		}
		trss[tr.UID] = &trState{status: s}
		return true
	}
	trss[tr.UID] = &trState{status: s}
	return true
}

func (i *interceptor) Process(ctx context.Context, req *v1beta1.InterceptorRequest) *v1beta1.InterceptorResponse {
	logger := logging.FromContext(ctx)
	ceType, ok := req.Header["Ce-Type"]
	if !ok || len(ceType) == 0 {
		logger.Warnw("Interceptor received empty cloud event; skipping")
		return interceptors.Fail(codes.InvalidArgument, "Only cloud event is allowed")
	}
	var ce cloudevent.TektonCloudEventData
	if err := json.NewDecoder(strings.NewReader(req.Body)).Decode(&ce); err != nil {
		logger.Errorw("Interceptor failed to unmarshal cloud event", zap.Error(err))
		return interceptors.Fail(codes.InvalidArgument, "Cloud event is malformed")
	}
	if !checkRunStatusChanged(ceType[0], &ce) {
		es++
		return &v1beta1.InterceptorResponse{
			Continue: false,
			Status: v1beta1.Status{
				Code:    codes.OK,
				Message: fmt.Sprintf("Events skipped: %d", es),
			},
		}
	}
	err := i.service.Sync(ctx, ceType[0], &ce)
	if err != nil {
		logger.Errorw("Interceptor failed to sync status", zap.Error(err))
		return interceptors.Fail(codes.Internal, "Unable to sync status")
	}
	return &v1beta1.InterceptorResponse{
		Continue: false,
		Status: v1beta1.Status{
			Code:    codes.OK,
			Message: fmt.Sprintf("Events processed: %d, Events skipped: %d", len(trss), es),
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
