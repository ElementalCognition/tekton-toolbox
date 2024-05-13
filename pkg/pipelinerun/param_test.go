package pipelinerun

import (
	"context"
	"testing"

	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineresolver"
	"github.com/stretchr/testify/assert"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func TestParam_Reconcile_StringRef(t *testing.T) {
	r, err := pipelineresolver.NewCelResolver()
	assert.Nil(t, err)
	ctx := pipelineresolver.WithResolver(context.TODO(), r)
	p := &Param{
		Param: v1.Param{
			Value: v1.ParamValue{
				Type:      v1.ParamTypeString,
				StringVal: "body.repository.clone_url",
			},
		},
	}
	req := &pipelineresolver.Metadata{
		Body: map[string]interface{}{
			"repository": map[string]interface{}{
				"clone_url": "foo",
			},
		},
	}
	err = p.Reconcile(ctx, req)
	assert.Nil(t, err)
	assert.Equal(t, "foo", p.Value.StringVal)
}

func TestParam_Reconcile_ArrayRef(t *testing.T) {
	r, err := pipelineresolver.NewCelResolver()
	assert.Nil(t, err)
	ctx := pipelineresolver.WithResolver(context.TODO(), r)
	p := &Param{
		Param: v1.Param{
			Value: v1.ParamValue{
				Type: v1.ParamTypeArray,
				ArrayVal: []string{
					"body.repository.clone_url",
				},
			},
		},
	}
	req := &pipelineresolver.Metadata{
		Body: map[string]interface{}{
			"repository": map[string]interface{}{
				"clone_url": "foo",
			},
		},
	}
	err = p.Reconcile(ctx, req)
	assert.Nil(t, err)
	assert.Equal(t, []string{
		"foo",
	}, p.Value.ArrayVal)
}

func TestParam_Reconcile_StringDefault(t *testing.T) {
	r, err := pipelineresolver.NewCelResolver()
	assert.Nil(t, err)
	ctx := pipelineresolver.WithResolver(context.TODO(), r)
	p := &Param{
		Param: v1.Param{
			Value: v1.ParamValue{
				Type:      v1.ParamTypeString,
				StringVal: "foo",
			},
		},
	}
	req := &pipelineresolver.Metadata{
		Body: map[string]interface{}{},
	}
	err = p.Reconcile(ctx, req)
	assert.Nil(t, err)
	assert.Equal(t, "foo", p.Value.StringVal)
}

func TestParam_Reconcile_ArrayDefault(t *testing.T) {
	r, err := pipelineresolver.NewCelResolver()
	assert.Nil(t, err)
	ctx := pipelineresolver.WithResolver(context.TODO(), r)
	p := &Param{
		Param: v1.Param{
			Value: v1.ParamValue{
				Type: v1.ParamTypeArray,
				ArrayVal: []string{
					"foo",
					"bar",
				},
			},
		},
	}
	req := &pipelineresolver.Metadata{
		Body: map[string]interface{}{},
	}
	err = p.Reconcile(ctx, req)
	assert.Nil(t, err)
	assert.Equal(t, []string{
		"foo",
		"bar",
	}, p.Value.ArrayVal)
}
