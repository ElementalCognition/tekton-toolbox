package githubstatussync

import (
	"fmt"
	"strings"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/reconciler/events/cloudevent"
)

var (
	checkRunStatusQueued     = "queued"
	checkRunStatusInProgress = "in_progress"
	checkRunStatusCompleted  = "completed"
)

// Subset of check conclusions when check has "completed" status.
// https://docs.github.com/en/rest/guides/using-the-rest-api-to-interact-with-checks?apiVersion=2022-11-28#about-check-suites
var (
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

func getFailureConclusion(trs v1beta1.TaskRunStatus) *string {
	var failureConclusion *string
	var reason string
	var message string

	// TODO hardcoding the first condition for now. Probably should be done better
	fmt.Printf("TaskRun status: %+v\n", trs.GetConditions()[0].Reason)
	reason = trs.GetConditions()[0].Reason
	message = trs.GetConditions()[0].Message
	switch reason {
	case v1beta1.TaskRunReasonCancelled.String():
		// It seems to be pretty hard to actually get a clear TimedOut reason for a task.
		// During tests I was unable to get it without PipelineRun time out cancelling it first.
		// So we somewhat hack around this problem by checking the message and deciding based on it.
		if strings.Contains(
			message,
			"TaskRun cancelled as the PipelineRun it belongs to has timed out.",
		) {
			failureConclusion = &checkRunConclusionTimedOut
		} else {
			failureConclusion = &checkRunConclusionCancelled
		}
	case v1beta1.TaskRunReasonTimedOut.String():
		failureConclusion = &checkRunConclusionTimedOut
	default:
		failureConclusion = &checkRunConclusionFailure
	}
	return failureConclusion
}

func status(eventType string, tr *v1beta1.TaskRun) (string, *string) {
	var status string
	var conclusion *string
	switch eventType {
	case cloudevent.TaskRunUnknownEventV1.String(), cloudevent.TaskRunStartedEventV1.String():
		status = checkRunStatusQueued
	case cloudevent.TaskRunRunningEventV1.String():
		status = checkRunStatusInProgress
	case cloudevent.TaskRunSuccessfulEventV1.String():
		status = checkRunStatusCompleted
		conclusion = &checkRunConclusionSuccess
	case cloudevent.TaskRunFailedEventV1.String():
		status = checkRunStatusCompleted
		if hasOptionalMarker(tr.Spec.Params) {
			conclusion = &checkRunConclusionNeutral
		} else {
			conclusion = getFailureConclusion(tr.Status)
		}
	default:
		status = checkRunStatusQueued
	}
	return status, conclusion
}
