package pipelinerun

import (
	"context"
	"fmt"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineresolver"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

type Param struct {
	v1.Param
}

var _ pipelineresolver.Reconciler = (*Param)(nil)

func (p *Param) Reconcile(ctx context.Context, metadata *pipelineresolver.Metadata) error {
	r, err := pipelineresolver.FromContext(ctx)
	if err != nil {
		return err
	}
	switch p.Value.Type {
	case v1.ParamTypeString:
		i, err := r.SafeValueOf(ctx, metadata, p.Value.StringVal)
		if err != nil {
			return err
		}
		p.Value.StringVal = fmt.Sprint(i)
	case v1.ParamTypeArray:
		for k, v := range p.Value.ArrayVal {
			i, err := r.SafeValueOf(ctx, metadata, v)
			if err != nil {
				return err
			}
			p.Value.ArrayVal[k] = fmt.Sprint(i)
		}
	}
	return nil
}

type ParamSlice []*Param

var _ pipelineresolver.Reconciler = (*ParamSlice)(nil)

func (s *ParamSlice) Reconcile(ctx context.Context, meta *pipelineresolver.Metadata) error {
	for _, p := range *s {
		if err := p.Reconcile(ctx, meta); err != nil {
			return err
		}
	}
	return nil
}
