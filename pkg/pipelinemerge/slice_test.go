package pipelinemerge

import (
	"testing"

	"github.com/imdario/mergo"
	"github.com/stretchr/testify/assert"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
)

func TestMergeSlice_SameValues(t *testing.T) {
	dst := &v1.PipelineRunSpec{
		Params: []v1.Param{
			{
				Name:  "param-1",
				Value: v1.ParamValue{},
			},
		},
	}
	src := &v1.PipelineRunSpec{
		Params: []v1.Param{
			{
				Name: "param-1",
				Value: v1.ParamValue{
					Type:      v1.ParamTypeString,
					StringVal: "param-1-value",
				},
			},
		},
	}
	err := mergo.Merge(dst, src, mergo.WithOverride, WithMergeSliceByName())
	assert.Nil(t, err)
	assert.Equal(t, v1.Params{
		v1.Param{
			Name: "param-1",
			Value: v1.ParamValue{
				Type:      v1.ParamTypeString,
				StringVal: "param-1-value",
			},
		},
	}, dst.Params)
}

func TestMergeSlice_DifferentValues(t *testing.T) {
	dst := &v1.PipelineRunSpec{
		Params: []v1.Param{
			{
				Name:  "param-1",
				Value: v1.ParamValue{},
			},
		},
	}
	src := &v1.PipelineRunSpec{
		Params: []v1.Param{
			{
				Name: "param-2",
				Value: v1.ParamValue{
					Type:      v1.ParamTypeString,
					StringVal: "param-2-value",
				},
			},
		},
	}
	err := mergo.Merge(dst, src, mergo.WithOverride, WithMergeSliceByName())
	assert.Nil(t, err)
	assert.Equal(t, v1.Params{
		v1.Param{
			Name:  "param-1",
			Value: v1.ParamValue{},
		},
		v1.Param{
			Name: "param-2",
			Value: v1.ParamValue{
				Type:      v1.ParamTypeString,
				StringVal: "param-2-value",
			},
		},
	}, dst.Params)
}

func TestMergeSlice_DuplicateValues(t *testing.T) {
	dst := &v1.PipelineRunSpec{
		Params: []v1.Param{
			{
				Name: "param-1",
				Value: v1.ParamValue{
					Type:      v1.ParamTypeString,
					StringVal: "param-1-value",
				},
			},
		},
	}
	src := &v1.PipelineRunSpec{
		Params: []v1.Param{
			{
				Name: "param-1",
				Value: v1.ParamValue{
					Type:      v1.ParamTypeString,
					StringVal: "param-1-value",
				},
			},
			{
				Name: "param-1",
				Value: v1.ParamValue{
					Type:      v1.ParamTypeString,
					StringVal: "param-1-new-value",
				},
			},
		},
	}
	err := mergo.Merge(dst, src, mergo.WithOverride, WithMergeSliceByName(mergo.WithOverride))
	assert.Nil(t, err)
	assert.Equal(t, v1.Params{
		v1.Param{
			Name: "param-1",
			Value: v1.ParamValue{
				Type:      v1.ParamTypeString,
				StringVal: "param-1-new-value",
			},
		},
	}, dst.Params)
}
