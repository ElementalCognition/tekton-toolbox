package githubstatussync

import (
	"github.com/tektoncd/pipeline/pkg/reconciler/events/cloudevent"
)

const (
	checkRunStatusQueued     = "queued"
	checkRunStatusInProgress = "in_progress"
	checkRunStatusCompleted  = "completed"
)

func status(eventType string) string {
	var status string

	switch eventType {
	case cloudevent.TaskRunUnknownEventV1.String(), cloudevent.TaskRunStartedEventV1.String():
		status = checkRunStatusQueued
	case cloudevent.TaskRunRunningEventV1.String():
		status = checkRunStatusInProgress
	case cloudevent.TaskRunSuccessfulEventV1.String(), cloudevent.TaskRunFailedEventV1.String():
		status = checkRunStatusCompleted
	default:
		status = checkRunStatusQueued
	}
	return status
}
