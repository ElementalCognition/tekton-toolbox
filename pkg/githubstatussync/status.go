package githubstatussync

import (
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
	checkRunConclusionSuccess = "success"
	// checkRunConclusionNeutral   = "neutral"
	checkRunConclusionTimedOut  = "timed_out"
	checkRunConclusionCancelled = "cancelled"
	checkRunConclusionFailure   = "failure"
)

func get_failure_conclusion(reason string) *string {
	var failureConclusion *string
	switch reason {
	case v1beta1.TaskRunReasonCancelled.String():
		failureConclusion = &checkRunConclusionCancelled
	case v1beta1.TaskRunReasonTimedOut.String():
		failureConclusion = &checkRunConclusionTimedOut
	default:
		failureConclusion = &checkRunConclusionFailure
	}
	return failureConclusion
}

func status(eventType string, reason string) (string, *string) {
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
		conclusion = get_failure_conclusion(reason)
	default:
		status = checkRunStatusQueued
	}
	return status, conclusion
}
