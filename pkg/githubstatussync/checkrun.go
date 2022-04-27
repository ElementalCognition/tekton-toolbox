package githubstatussync

import (
	"github.com/google/go-github/v43/github"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func checkRun(eventType string, tr *v1beta1.TaskRun) (*github.CreateCheckRunOptions, error) {
	url, err := detailsURL(tr)
	if err != nil {
		return nil, err
	}
	name, err := nameFor(tr)
	if err != nil {
		return nil, err
	}
	status, conclusion := status(eventType)
	completedAt := timestamp(tr.Status.CompletionTime)
	switch status {
	case checkRunStatusInProgress:
	case checkRunStatusQueued:
		completedAt = nil
	}
	ref := tr.Annotations[refKey.String()]
	return &github.CreateCheckRunOptions{
		ExternalID:  github.String(string(tr.UID)),
		Name:        name,
		Status:      github.String(status),
		Conclusion:  conclusion,
		HeadSHA:     ref,
		StartedAt:   timestamp(tr.Status.StartTime),
		CompletedAt: completedAt,
		DetailsURL:  github.String(url),
	}, nil
}
