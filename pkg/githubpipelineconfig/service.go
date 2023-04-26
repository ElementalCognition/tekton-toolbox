package githubpipelineconfig

import (
	"context"

	"github.com/ElementalCognition/tekton-toolbox/pkg/pipelineconfig"
	"github.com/ElementalCognition/tekton-toolbox/pkg/vcspipelineconfig"
	"github.com/google/go-github/v43/github"
	"go.uber.org/zap"
	"knative.dev/pkg/logging"
)

const configFile = ".tekton.yaml"

type service struct {
	githubClient *github.Client
}

var _ vcspipelineconfig.Service = (*service)(nil)

func (s *service) Get(ctx context.Context, owner, repo, ref string) (*pipelineconfig.Config, error) {
	logger := logging.FromContext(ctx)
	logger = logger.With(
		zap.String("owner", owner),
		zap.String("repository", repo),
	)
	logger.Infow("Service started fetch config")
	opts := &github.RepositoryContentGetOptions{Ref: ref}
	c, _, res, err := s.githubClient.Repositories.GetContents(ctx, owner, repo, configFile, opts)
	if err != nil {
		logger.Errorw("Service failed to fetch config",
			zap.String("responseStatus", res.Status),
			zap.Error(err),
		)
		return nil, err
	}
	content, err := c.GetContent()
	if err != nil {
		return nil, err
	}
	if err != nil {
		return nil, err
	}
	cfg := &pipelineconfig.Config{}
	err = cfg.UnmarshalYAML([]byte(content))
	return cfg, err
}

func NewService(
	githubClient *github.Client,
) vcspipelineconfig.Service {
	return &service{
		githubClient: githubClient,
	}
}
