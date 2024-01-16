package githubstatussync

import (
	"context"

	"github.com/ElementalCognition/tekton-toolbox/pkg/cloudeventsync"
	"github.com/google/go-github/v43/github"
	"github.com/tektoncd/pipeline/pkg/reconciler/events/cloudevent"
	"go.uber.org/zap"
	"knative.dev/pkg/logging"
)

type service struct {
	githubClient *github.Client
}

var _ cloudeventsync.Service = (*service)(nil)

func (s *service) Sync(
	ctx context.Context,
	eventType string,
	cloudEvent *cloudevent.TektonCloudEventData,
) error {
	logger := logging.FromContext(ctx)
	tr := cloudEvent.TaskRun
	if tr == nil {
		logger.Warnw("Service received unsupported cloud event; skipping")
		return nil
	}
	cro, err := checkRun(ctx, eventType, tr)
	if err != nil {
		return err
	}
	repoName := tr.Annotations[repoKey.String()]
	ownerName := tr.Annotations[ownerKey.String()]
	logger = logger.With(
		zap.String("event", eventType),
		zap.String("taskRun", tr.GetNamespacedName().String()),
		zap.String("repo", repoName),
		zap.String("owner", ownerName),
		zap.String("name", cro.Name),
		zap.Stringp("detailsUrl", cro.DetailsURL),
		zap.Stringp("status", cro.Status),
		zap.Timep("startedAt", time(cro.StartedAt)),
	)
	if *cro.Status == checkRunStatusCompleted {
		logger = logger.With(
			zap.Stringp("conclusion", cro.Conclusion),
			zap.Timep("completedAt", time(cro.CompletedAt)),
		)
	}
	logger.Infow("Service started sync status")
	cr, res, err := s.githubClient.Checks.CreateCheckRun(ctx, ownerName, repoName, *cro)
	if err != nil {
		logger.Errorw("Service failed to sync status",
			zap.String("responseStatus", res.Status),
			zap.Error(err),
		)
	} else {
		logger.Infow("Service finished sync status",
			zap.String("responseStatus", res.Status),
			zap.Stringp("externalId", cr.ExternalID),
			zap.Stringp("nodeId", cr.NodeID),
		)
	}
	return err
}

func NewService(
	githubClient *github.Client,
) cloudeventsync.Service {
	return &service{
		githubClient: githubClient,
	}
}
