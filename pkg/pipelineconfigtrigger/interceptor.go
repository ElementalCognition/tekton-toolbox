package pipelineconfigtrigger

import (
	"context"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineresolver"
	"github.com/ElementalCognition/tekton-toolbox/pkg/triggers"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"knative.dev/pkg/logging"
)

var _ v1beta1.InterceptorInterface = (*interceptor)(nil)

type interceptor struct {
	service  Service
	resolver pipelineresolver.Resolver
}

func (i *interceptor) Process(ctx context.Context, req *v1beta1.InterceptorRequest) *v1beta1.InterceptorResponse {
	logger := logging.FromContext(ctx)
	rw := triggers.InterceptorRequest(*req)
	body, err := rw.UnmarshalBody()
	if err != nil {
		logger.Errorw("Interceptor failed to unmarshal request", zap.Error(err))
		return interceptors.Fail(codes.Internal, "Request body is malformed")
	}
	cfg, err := rw.CurrentConfig()
	if err != nil {
		logger.Errorw("Interceptor failed to get current config", zap.Error(err))
		return interceptors.Fail(codes.InvalidArgument, "Unable to get current config")
	}
	prs, err := cfg.PipelineRuns(
		pipelineresolver.WithResolver(ctx, i.resolver),
		&pipelineresolver.Metadata{
			Header:     req.Header,
			Extensions: req.Extensions,
			Params:     req.InterceptorParams,
			Body:       body,
		})
	if err != nil {
		logger.Errorw("Interceptor failed to get pipeline runs from config", zap.Error(err))
		return interceptors.Fail(codes.Internal, "Unable to get pipeline runs from config")
	}
	err = i.service.Create(ctx, prs...)
	if err != nil {
		logger.Errorw("Interceptor failed to trigger pipeline runs", zap.Error(err))
		return interceptors.Fail(codes.Internal, "Unable to trigger pipeline runs")
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
	resolver pipelineresolver.Resolver,
) v1beta1.InterceptorInterface {
	return &interceptor{
		service:  service,
		resolver: resolver,
	}
}
