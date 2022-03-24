package pipelinerun

import (
	"context"
	"fmt"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelinemerge"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineresolver"
	"github.com/imdario/mergo"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PipelineRun struct {
	v1beta1.PipelineRunSpec
	Name     string `json:"name,omitempty"`
	Metadata `json:"metadata,omitempty"`
	Params   ParamSlice `json:"params,omitempty"`
}

var _ pipelineresolver.Reconciler = (*PipelineRun)(nil)

func (p *PipelineRun) Merge(d *PipelineRun) error {
	return mergo.Merge(p, d, pipelinemerge.DefaultOptions...)
}

func (p *PipelineRun) MergeAll(ds ...*PipelineRun) error {
	for _, d := range ds {
		err := p.Merge(d)
		if err != nil {
			return err
		}
	}
	return nil
}

func (p *PipelineRun) Reconcile(ctx context.Context, meta *pipelineresolver.Metadata) error {
	err := p.Metadata.Reconcile(ctx, meta)
	if err != nil {
		return err
	}
	return p.Params.Reconcile(ctx, meta)
}

func (p *PipelineRun) PipelineRun() (*v1beta1.PipelineRun, error) {
	name := p.Name
	if p.PipelineRef != nil && len(p.PipelineRef.Name) > 0 {
		name = p.PipelineRef.Name
	}
	meta := p.Metadata.DeepCopy()
	meta.GenerateName = fmt.Sprintf("%s-run-", name)
	var params []v1beta1.Param
	for _, p := range p.Params {
		params = append(params, *p.Param.DeepCopy())
	}
	spec := p.PipelineRunSpec.DeepCopy()
	spec.Params = params
	return &v1beta1.PipelineRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PipelineRun",
			APIVersion: "tekton.dev/v1beta1",
		},
		ObjectMeta: *meta,
		Spec:       *spec,
	}, nil
}

type PipelineSlice []PipelineRun
