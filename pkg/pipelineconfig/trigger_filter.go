package pipelineconfig

import (
	"context"
	"fmt"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineresolver"
)

type TriggerFilter string

func (f TriggerFilter) Match(ctx context.Context, meta *pipelineresolver.Metadata) (bool, error) {
	r, err := pipelineresolver.FromContext(ctx)
	if err != nil {
		return false, err
	}
	v, err := r.ValueOf(ctx, meta, string(f))
	if err != nil {
		return false, err
	}
	m, ok := v.(bool)
	if !ok {
		return false, fmt.Errorf("unable to convert filter '%v' value from '%T' to 'boolean'", string(f), v)
	}
	return m, nil
}
