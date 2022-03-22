package pipelineconfig

import "github.com/ElementalCognition/tekton-toolbox/pkg/pipelinerun"

type Trigger struct {
	Name      string                    `json:"name,omitempty"`
	Filter    TriggerFilter             `json:"filter,omitempty"`
	Defaults  pipelinerun.PipelineRun   `json:"defaults,omitempty"`
	Pipelines pipelinerun.PipelineSlice `json:"pipelines,omitempty"`
}

type TriggerSlice []Trigger
