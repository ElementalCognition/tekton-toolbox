package githubstatussync

import (
	"fmt"

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

func hasOptionalMarker(params v1beta1.Params) bool {
	for _, p := range params {
		if p.Name == "optional-task" && p.Value.StringVal == "true" {
			return true
		}
	}
	return false
}

func getFailureConclusion(tr *v1beta1.TaskRun) *string {
	var failureConclusion *string
	var reason string
	var message string

	fmt.Printf("TaskRun params: %+v\n", tr.Spec.Params)
	fmt.Printf("TaskRun status: %+v\n", tr.Status.GetConditions()[0].Reason)
	// TODO hardcoding the first condition for now. Probably should be done better
	reason = tr.Status.GetConditions()[0].Reason
	message = tr.Status.GetConditions()[0].Message
	if hasOptionalMarker(tr.Spec.Params) {
		return &checkRunConclusionNeutral
	} else {
		switch reason {
		case v1beta1.TaskRunReasonCancelled.String():
			if message == "TaskRun cancelled as the PipelineRun it belongs to has timed out." {
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
		conclusion = getFailureConclusion(tr)
	default:
		status = checkRunStatusQueued
	}
	return status, conclusion
}
