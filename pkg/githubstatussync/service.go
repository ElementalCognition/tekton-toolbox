package githubstatussync

import (
	"context"

	"github.com/ElementalCognition/tekton-toolbox/pkg/cloudeventsync"
	"github.com/google/go-github/v43/github"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
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
		logger.Warnw("Service received unsupported cloud event, nil TaskRun; skipping")
		return nil
	}
	trV1 := new(v1.TaskRun)
	if err := trV1.ConvertTo(ctx, tr); err != nil {
		logger.Warnf("Service unable to convert cloud event from v1beta1 to v1; err: %v", err)
		return nil
	}
	cro, err := checkRun(ctx, eventType, trV1)
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
		logger = logger.With(zap.Stringp("conclusion", cro.Conclusion),
			zap.Timep("completedAt", time(cro.CompletedAt)))
	}
	logger.Infow("Service started sync status")
	// This bit might be quite confusing. While the API reference states that this should be use for creation only,
	// actually running it again while keeping the same name (and only name) will "overwrite" check run.
	// In practice it means that the new check run will be created, but github will only use the latest.
	// All the previous check runs will be still in place and reachable via direct URL, but they won't affect the check suite.
	// In theory, we should use UpdateCheckRun API (or method) for everything that is not just queued.
	//
	// But this proves to be quite cumbersome as we need to "get" checks for a ref, then find the correct one by name, ExternalID
	// or somethig else (or a combination) to then actually update. It seems it doesn't actually provide any real benifit though.
	cr, res, err := s.githubClient.Checks.CreateCheckRun(ctx, ownerName, repoName, *cro)
	if err != nil {
		keyAndVals := []any{zap.Error(err)}
		if res != nil {
			keyAndVals = append(keyAndVals, zap.String("responseStatus", res.Status))
		}
		logger.Errorw("Service failed to sync status", keyAndVals...)
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
