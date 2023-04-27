package pipelineconfig

import (
	"context"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineresolver"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestTriggerFilter_Match_True(t *testing.T) {
	r, err := pipelineresolver.NewCelResolver()
	assert.Nil(t, err)
	ctx := pipelineresolver.WithResolver(context.TODO(), r)
	req := &pipelineresolver.Metadata{
		Body: map[string]interface{}{
			"action": "opened",
		},
	}
	f := TriggerFilter(`body.action in ["opened", "synchronize", "reopened"]`)
	m, err := f.Match(ctx, req)
	assert.Nil(t, err)
	assert.True(t, m)
}

func TestTriggerFilter_Match_False(t *testing.T) {
	r, err := pipelineresolver.NewCelResolver()
	assert.Nil(t, err)
	ctx := pipelineresolver.WithResolver(context.TODO(), r)
	req := &pipelineresolver.Metadata{
		Body: map[string]interface{}{
			"ref": "refs/heads/master",
		},
	}
	f := TriggerFilter(`body.ref == "refs/heads/main"`)
	m, err := f.Match(ctx, req)
	assert.Nil(t, err)
	assert.False(t, m)
}

func TestTriggerFilter_Match_UnsupportedType(t *testing.T) {
	r, err := pipelineresolver.NewCelResolver()
	assert.Nil(t, err)
	ctx := pipelineresolver.WithResolver(context.TODO(), r)
	req := &pipelineresolver.Metadata{
		Body: map[string]interface{}{
			"ref": "refs/heads/master",
		},
	}
	f := TriggerFilter("body.ref")
	m, err := f.Match(ctx, req)
	assert.NotNil(t, err)
	assert.False(t, m)
}
