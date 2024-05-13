package pipelineconfigtrigger

import (
	"context"
	"github.com/hashicorp/go-multierror"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	"go.uber.org/zap"
	"gopkg.in/go-playground/pool.v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/logging"
)

type Service interface {
	Create(ctx context.Context, pipelineRuns ...*v1.PipelineRun) error
}

type service struct {
	tektonClient versioned.Interface
	pool         pool.Pool
}

var _ Service = (*service)(nil)

func (s *service) create(ctx context.Context, pipelineRun *v1.PipelineRun) func(wu pool.WorkUnit) (interface{}, error) {
	return func(wu pool.WorkUnit) (interface{}, error) {
		if wu.IsCancelled() {
			return nil, nil
		}
		logger := logging.FromContext(ctx)
		logger.Infow("Service started pipeline run",
			zap.String("namespace", pipelineRun.Namespace),
			zap.String("name", pipelineRun.Name),
			zap.String("generateName", pipelineRun.GenerateName),
			zap.Any("labels", pipelineRun.Labels),
			zap.Any("annotations", pipelineRun.Annotations),
		)
		return s.tektonClient.TektonV1().
			PipelineRuns(pipelineRun.Namespace).
			Create(ctx, pipelineRun, metav1.CreateOptions{})
	}
}

func (s *service) Create(ctx context.Context, pipelineRuns ...*v1.PipelineRun) error {
	me := new(multierror.Error)
	batch := s.pool.Batch()
	for _, pipelineRun := range pipelineRuns {
		batch.Queue(s.create(ctx, pipelineRun))
	}
	batch.QueueComplete()
	for r := range batch.Results() {
		if r.Error() != nil {
			me = multierror.Append(me, r.Error())
		}
	}
	return me.ErrorOrNil()
}

func NewService(
	tektonClient versioned.Interface,
	pool pool.Pool,
) Service {
	return &service{
		tektonClient: tektonClient,
		pool:         pool,
	}
}
