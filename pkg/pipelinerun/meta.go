package pipelinerun

import (
	"context"
	"fmt"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineresolver"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Metadata struct {
	metav1.ObjectMeta
}

var _ pipelineresolver.Reconciler = (*Metadata)(nil)

func (m *Metadata) Reconcile(ctx context.Context, meta *pipelineresolver.Metadata) error {
	err := MetadataMap(m.Labels).Reconcile(ctx, meta)
	if err != nil {
		return err
	}
	err = MetadataMap(m.Annotations).Reconcile(ctx, meta)
	if err != nil {
		return err
	}
	return nil
}

type MetadataMap map[string]string

var _ pipelineresolver.Reconciler = (*MetadataMap)(nil)

func (m MetadataMap) Reconcile(ctx context.Context, meta *pipelineresolver.Metadata) error {
	r, err := pipelineresolver.FromContext(ctx)
	if err != nil {
		return err
	}
	for k, v := range m {
		i, err := r.SafeValueOf(ctx, meta, v)
		if err != nil {
			return err
		}
		m[k] = fmt.Sprint(i)
	}
	return nil
}
