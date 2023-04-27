package kubepipelineconfig

import (
	"context"
	"fmt"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineconfig"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineresolver"
	"github.com/ElementalCognition/tekton-toolbox/pkg/triggers"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"knative.dev/pkg/logging"
)

const (
	nsParamKey   = "namespace"
	nameParamKey = "name"
)

var _ v1beta1.InterceptorInterface = (*interceptor)(nil)

type interceptor struct {
	service  Service
	resolver pipelineresolver.Resolver
}

func (i *interceptor) valueOf(ctx context.Context, meta *pipelineresolver.Metadata, expr string) (string, error) {
	val, err := i.resolver.SafeValueOf(ctx, meta, expr)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(val), nil
}

func (i *interceptor) findParam(ctx context.Context, meta *pipelineresolver.Metadata, name string) (string, error) {
	refVal, ok := meta.Params[name]
	if !ok {
		return "", fmt.Errorf("%s param does not exist", name)
	}
	return i.valueOf(ctx, meta, fmt.Sprint(refVal))
}

func (i *interceptor) Process(ctx context.Context, req *v1beta1.InterceptorRequest) *v1beta1.InterceptorResponse {
	logger := logging.FromContext(ctx)
	rw := triggers.InterceptorRequest(*req)
	body, err := rw.UnmarshalBody()
	if err != nil {
		logger.Errorw("Interceptor failed to unmarshal request body", zap.Error(err))
		return interceptors.Fail(codes.InvalidArgument, "Request body is malformed")
	}
	meta := &pipelineresolver.Metadata{
		Header:     req.Header,
		Extensions: req.Extensions,
		Params:     req.InterceptorParams,
		Body:       body,
	}
	ns, err := i.findParam(ctx, meta, nsParamKey)
	if err != nil {
		logger.Errorw("Interceptor failed to get `namespace` parameter", zap.Error(err))
		return interceptors.Fail(codes.InvalidArgument, "`namespace` parameter is required")
	}
	name, err := i.findParam(ctx, meta, nameParamKey)
	if err != nil {
		logger.Errorw("Interceptor failed to get `name` parameter", zap.Error(err))
		return interceptors.Fail(codes.InvalidArgument, "`name` parameter is required")
	}
	prevCfg, err := i.service.Get(ctx, ns, name)
	if err != nil {
		logger.Errorw("Interceptor failed to fetch config", zap.Error(err))
		return interceptors.Fail(codes.Internal, "Unable to fetch config")
	}
	nextCfg, err := rw.MergeConfig(prevCfg)
	if err != nil {
		logger.Errorw("Interceptor failed to merge config", zap.Error(err))
		return interceptors.Fail(codes.Internal, "Unable to merge config")
	}
	buf, err := nextCfg.MarshalJSON()
	if err != nil {
		logger.Errorw("Interceptor failed to marshal config", zap.Error(err))
		return interceptors.Fail(codes.Internal, "Unable to marshal config")
	}
	return &v1beta1.InterceptorResponse{
		Continue: true,
		Extensions: map[string]interface{}{
			pipelineconfig.ConfigKey: string(buf),
		},
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
