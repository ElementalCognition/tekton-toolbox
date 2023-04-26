package vcspipelineconfig

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
	ownerLoginKey = "owner"
	repoNameKey   = "repo"
	refParamKey   = "ref"
)

type interceptor struct {
	service  Service
	resolver pipelineresolver.Resolver
}

var _ v1beta1.InterceptorInterface = (*interceptor)(nil)

func (i *interceptor) valueOf(ctx context.Context, meta *pipelineresolver.Metadata, expr string) (string, error) {
	val, err := i.resolver.SafeValueOf(ctx, meta, expr)
	if err != nil {
		return "", err
	}
	return fmt.Sprint(val), nil
}

func (i *interceptor) findParam(ctx context.Context, meta *pipelineresolver.Metadata, name string) (string, error) {
	val, ok := meta.Params[name]
	if !ok {
		return "", fmt.Errorf("%s param does not exist", name)
	}
	return i.valueOf(ctx, meta, fmt.Sprint(val))
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
	owner, err := i.findParam(ctx, meta, ownerLoginKey)
	if err != nil {
		logger.Errorw("Interceptor failed to get `owner` parameter", zap.Error(err))
		return interceptors.Fail(codes.InvalidArgument, "`owner` parameter is required")
	}
	repo, err := i.findParam(ctx, meta, repoNameKey)
	if err != nil {
		logger.Errorw("Interceptor failed to get `repo` parameter", zap.Error(err))
		return interceptors.Fail(codes.InvalidArgument, "`repo` parameter is required")
	}
	ref, err := i.findParam(ctx, meta, refParamKey)
	if err != nil {
		logger.Errorw("Interceptor failed to get `ref` parameter", zap.Error(err))
		return interceptors.Fail(codes.InvalidArgument, "`ref` parameter is required")
	}
	if err != nil {
		logger.Errorw("Interceptor failed to find repository", zap.Error(err))
		return interceptors.Fail(codes.InvalidArgument, "Unable to find repository")
	}
	prevCfg, err := i.service.Get(ctx, owner, repo, ref)
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
