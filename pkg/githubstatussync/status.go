package githubstatussync

import "github.com/tektoncd/pipeline/pkg/reconciler/events/cloudevent"

var (
	checkRunStatusQueued      = "queued"
	checkRunStatusInProgress  = "in_progress"
	checkRunStatusCompleted   = "completed"
	checkRunConclusionSuccess = "success"
	checkRunConclusionFailure = "failure"
)

func status(eventType string) (string, *string) {
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
		conclusion = &checkRunConclusionFailure
	default:
		status = checkRunStatusQueued
	}
	return status, conclusion
}
