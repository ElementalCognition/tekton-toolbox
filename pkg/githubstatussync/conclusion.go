package githubstatussync

import (
	"context"
	"strings"

	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"github.com/tektoncd/pipeline/pkg/reconciler/events/cloudevent"
	"knative.dev/pkg/logging"
)

const optionalMarker = "optional-task"

// Subset of check conclusions when check has "completed" status.
// https://docs.github.com/en/rest/guides/using-the-rest-api-to-interact-with-checks?apiVersion=2022-11-28#about-check-suites
const (
	checkRunConclusionSuccess   = "success"
	checkRunConclusionNeutral   = "neutral"
	checkRunConclusionTimedOut  = "timed_out"
	checkRunConclusionCancelled = "cancelled"
	checkRunConclusionFailure   = "failure"
)

// Verify if the paramater list for a TaskRun contains this variable set to true (string).
func hasOptionalMarker(trp v1.Params) bool {
	for _, p := range trp {
		if p.Name == optionalMarker && p.Value.StringVal == "true" {
			return true
		}
	}
	return false
}

// Resolve specific reason for failure based on the TaskRunStatus via Conditions (depends on Reason and Message).
func getFailureConclusion(ctx context.Context, trs v1.TaskRunStatus) string {
	logger := logging.FromContext(ctx)

	trsLen := len(trs.GetConditions())
	if trsLen == 0 {
		logger.Errorw("Received empty conditions, can't determine status %v", trs)
		return checkRunConclusionFailure
	}
	logger.Debugw("TaskRun status: %+v", trs.GetConditions())

	var failureConclusion, reason, message string

	// Was unable to find in which order conditions are added.
	// Current assumption would be that the latest condition will be last.
	reason = trs.GetConditions()[trsLen-1].GetReason()
	message = trs.GetConditions()[trsLen-1].GetMessage()
	switch reason {
	case v1.TaskRunReasonCancelled.String():
		// It seems to be pretty hard to actually get a clear TimedOut reason for a task.
		// During tests I was unable to get it without PipelineRun time out cancelling it first.
		// So we somewhat hack around this problem by checking the message and deciding based on it.
		failureConclusion = checkRunConclusionCancelled
		if strings.Contains(
			message,
			"TaskRun cancelled as the PipelineRun it belongs to has timed out.",
		) {
			logger.Debugw("Detected PipelineRun timeout, counting as general timeout")
			failureConclusion = checkRunConclusionTimedOut
		}
	case v1.TaskRunReasonTimedOut.String():
		failureConclusion = checkRunConclusionTimedOut
	default:
		failureConclusion = checkRunConclusionFailure
	}

	logger.Debugw("Resolved conclusion: %s", failureConclusion)
	return failureConclusion
}

// Resolve github resolveConclusion for completed TaskRuns.
func resolveConclusion(ctx context.Context, eventType string, tr *v1.TaskRun) string {
	var conclusion string
	logger := logging.FromContext(ctx)

	if eventType == cloudevent.TaskRunSuccessfulEventV1.String() {
		return checkRunConclusionSuccess
	}

	if hasOptionalMarker(tr.Spec.Params) {
		logger.Infow("Found optional marker: %+v", tr.Spec.Params)
		conclusion = checkRunConclusionNeutral
	} else {
		conclusion = getFailureConclusion(ctx, tr.Status)
	}

	return conclusion
}
