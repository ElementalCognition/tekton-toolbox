package githubstatussync

import (
	"context"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/events/cloudevent"
	"knative.dev/pkg/logging"
)

const (
	checkRunStatusQueued     = "queued"
	checkRunStatusInProgress = "in_progress"
	checkRunStatusCompleted  = "completed"
)

// Subset of check conclusions when check has "completed" status.
// https://docs.github.com/en/rest/guides/using-the-rest-api-to-interact-with-checks?apiVersion=2022-11-28#about-check-suites
const (
	checkRunConclusionSuccess   = "success"
	checkRunConclusionNeutral   = "neutral"
	checkRunConclusionTimedOut  = "timed_out"
	checkRunConclusionCancelled = "cancelled"
	checkRunConclusionFailure   = "failure"
)

var optionalMarker = "optional-task"

func hasOptionalMarker(trp v1beta1.Params) bool {
	for _, p := range trp {
		if p.Name == optionalMarker && p.Value.StringVal == "true" {
			return true
		}
	}
	return false
}

// getFailureConclusion
// Resolve specific reason for failure based on the TaskRunStatus via Conditions (depends on Reason and Message).
func getFailureConclusion(ctx context.Context, trs v1beta1.TaskRunStatus) string {
	var failureConclusion, reason, message string

	logger := logging.FromContext(ctx)

	trsLen := len(trs.GetConditions())
	if trsLen == 0 {
		logger.Errorw("Received empty conditions, can't determine status %v", trs)
		return checkRunConclusionFailure
	}
	logger.Warnf("TaskRun status: %+v", trs.GetConditions())

	// TODO Find how tekton orders conditions.
	reason = trs.GetConditions()[trsLen-1].GetReason()
	message = trs.GetConditions()[trsLen-1].GetMessage()
	switch reason {
	case v1beta1.TaskRunReasonCancelled.String():
		// It seems to be pretty hard to actually get a clear TimedOut reason for a task.
		// During tests I was unable to get it without PipelineRun time out cancelling it first.
		// So we somewhat hack around this problem by checking the message and deciding based on it.
		failureConclusion = checkRunConclusionCancelled
		if strings.Contains(
			message,
			"TaskRun cancelled as the PipelineRun it belongs to has timed out.",
		) {
			logger.Warnf("Detected PipelineRun timeout, counting as general timeout")
			failureConclusion = checkRunConclusionTimedOut
		}
	case v1beta1.TaskRunReasonTimedOut.String():
		failureConclusion = checkRunConclusionTimedOut
	default:
		failureConclusion = checkRunConclusionFailure
	}

	logger.Warnf("Resolved conclusion: %s", failureConclusion)
	return failureConclusion
}

func status(ctx context.Context, eventType string, tr *v1beta1.TaskRun) (string, string) {
	var status, conclusion string

	logger := logging.FromContext(ctx)

	switch eventType {
	case cloudevent.TaskRunUnknownEventV1.String(), cloudevent.TaskRunStartedEventV1.String():
		status = checkRunStatusQueued
	case cloudevent.TaskRunRunningEventV1.String():
		status = checkRunStatusInProgress
	case cloudevent.TaskRunSuccessfulEventV1.String():
		status = checkRunStatusCompleted
		conclusion = checkRunConclusionSuccess
	case cloudevent.TaskRunFailedEventV1.String():
		status = checkRunStatusCompleted
		logger.Warnf("Failed event: %+v", tr)
		if hasOptionalMarker(tr.Spec.Params) {
			logger.Warnf("Found optional marker: %+v", tr.Spec.Params)
			conclusion = checkRunConclusionNeutral
		} else {
			conclusion = getFailureConclusion(ctx, tr.Status)
		}
	default:
		status = checkRunStatusQueued
	}
	return status, conclusion
}
