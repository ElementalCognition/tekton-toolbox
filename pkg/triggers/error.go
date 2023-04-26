package triggers

import "errors"

var (
	ErrPipelineConfigNotFound  = errors.New("config does not exist")
	ErrPipelineConfigMalformed = errors.New("config is malformed")
)
