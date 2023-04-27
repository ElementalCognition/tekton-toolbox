package triggers

import (
	"encoding/json"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineconfig"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
)

type InterceptorRequest v1beta1.InterceptorRequest

func (tr *InterceptorRequest) CurrentConfig() (*pipelineconfig.Config, error) {
	v, ok := tr.Extensions[pipelineconfig.ConfigKey]
	if !ok {
		return nil, ErrPipelineConfigNotFound
	}
	s, ok := v.(string)
	if !ok {
		return nil, ErrPipelineConfigMalformed
	}
	c := &pipelineconfig.Config{}
	err := c.UnmarshalJSON([]byte(s))
	return c, err
}

func (tr *InterceptorRequest) MergeConfig(cfg *pipelineconfig.Config) (*pipelineconfig.Config, error) {
	c, err := tr.CurrentConfig()
	if err != nil {
		switch err {
		case ErrPipelineConfigNotFound, ErrPipelineConfigMalformed:
			c = &pipelineconfig.Config{}
		default:
			return nil, err
		}
	}
	err = c.Merge(cfg)
	return c, err
}

func (tr *InterceptorRequest) UnmarshalBody() (map[string]interface{}, error) {
	var body map[string]interface{}
	err := json.Unmarshal([]byte(tr.Body), &body)
	return body, err
}
