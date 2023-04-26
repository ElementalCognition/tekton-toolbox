package vcspipelineconfig

import (
	"context"
	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineconfig"
)

type Service interface {
	Get(ctx context.Context, owner, repo, ref string) (*pipelineconfig.Config, error)
}
