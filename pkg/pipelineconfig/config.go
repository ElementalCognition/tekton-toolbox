package pipelineconfig

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelinemerge"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineresolver"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelinerun"
	"github.com/imdario/mergo"
	v1pipeline "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/yaml"
)

const ConfigKey = "pipeline-config"

var _ json.Unmarshaler = (*Config)(nil)
var _ json.Marshaler = (*Config)(nil)

type Config struct {
	Defaults pipelinerun.PipelineRun `json:"defaults,omitempty" yaml:"defaults,omitempty"`
	Triggers TriggerSlice            `json:"triggers,omitempty" yaml:"triggers,omitempty"`
}

func (c *Config) Merge(s *Config) error {
	return mergo.Merge(c, s, pipelinemerge.DefaultOptions...)
}

func (c *Config) toPipelineRun(ctx context.Context, meta *pipelineresolver.Metadata, p ...*pipelinerun.PipelineRun) (*v1pipeline.PipelineRun, error) {
	tr := &pipelinerun.PipelineRun{}
	err := tr.MergeAll(p...)
	if err != nil {
		return nil, err
	}
	err = tr.Reconcile(ctx, meta)
	if err != nil {
		return nil, err
	}
	return tr.PipelineRun()
}

func (c *Config) PipelineRuns(ctx context.Context, meta *pipelineresolver.Metadata) ([]*v1pipeline.PipelineRun, error) {
	var prs []*v1pipeline.PipelineRun
	for _, t := range c.Triggers {
		ok, err := t.Filter.Match(ctx, meta)
		if err != nil {
			return nil, err
		}
		if !ok {
			continue
		}
		for _, p := range t.Pipelines {
			pr, err := c.toPipelineRun(ctx, meta, &c.Defaults, &t.Defaults, &p)
			if err != nil {
				return nil, err
			}
			prs = append(prs, pr)
		}
	}
	return prs, nil
}

func (c *Config) UnmarshalJSON(data []byte) error {
	type TriggerConfigJSON Config
	var t TriggerConfigJSON
	err := json.Unmarshal(data, &t)
	if err != nil {
		return err
	}
	*c = Config(t)
	return nil
}

func (c *Config) MarshalJSON() ([]byte, error) {
	type TriggerConfigJSON Config
	t := TriggerConfigJSON(*c)
	return json.Marshal(&t)
}

func (c *Config) UnmarshalYAML(data []byte) error {
	return yaml.UnmarshalStrict(data, &c)
}

func (c *Config) UnmarshalConfigMapYAML(cm *v1.ConfigMap) error {
	s, ok := cm.Data["config.yaml"]
	if !ok {
		return errors.New("unable to get config data from config map")
	}
	return c.UnmarshalYAML([]byte(s))
}
