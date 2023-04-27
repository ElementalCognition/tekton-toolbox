package pipelinemerge

import "github.com/imdario/mergo"

var DefaultOptions = []func(*mergo.Config){
	mergo.WithOverride,
	WithMergeSliceByName(mergo.WithOverride),
}
