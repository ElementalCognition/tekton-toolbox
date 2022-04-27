package githubstatussync

import (
	"fmt"

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

	var chkRunAnno []*github.CheckRunAnnotation

	for _, v := range tr.Status.Steps {
		var s string
		switch v.Terminated.Reason {
		case "Completed":
			s = "notice"
		case "Failed":
			s = "failure"
		case "TaskRunCancelled":
			s = "warning"
		default:
			s = "notice"
		}
		chkRunAnno = append(chkRunAnno, &github.CheckRunAnnotation{
			Path:            github.String(fmt.Sprintf("https://tekton-log.src.ec.ai/logs/%s/%s/%s", tr.Namespace, tr.Status.PodName, v.ContainerName)), // Dummy file name, required item.
			StartLine:       github.Int(1),                                                                                                              // Dummy int, required item.
			EndLine:         github.Int(1),                                                                                                              // Dummy int, required item.
			AnnotationLevel: github.String(s),                                                                                                           // Can be one of notice, warning, or failure.
			Title:           github.String(v.Name),
			// TODO: url must be removed and placed as an annotation in kube-pipeline config.
			Message:    github.String(fmt.Sprintf("https://tekton-log.src.ec.ai/logs/%s/%s/%s", tr.Namespace, tr.Status.PodName, v.ContainerName)),
			RawDetails: github.String(v.Terminated.Message),
		})
	}

	output := &github.CheckRunOutput{
		Title:       github.String("Steps details"),
		Summary:     github.String("Summary will be here"),
		Text:        github.String(""),
		Annotations: chkRunAnno,
	}

	return &github.CreateCheckRunOptions{
		ExternalID:  github.String(string(tr.UID)),
		Name:        name,
		Status:      github.String(status),
		Conclusion:  conclusion,
		HeadSHA:     ref,
		StartedAt:   timestamp(tr.Status.StartTime),
		CompletedAt: completedAt,
		DetailsURL:  github.String(url),
		Output:      output,
	}, nil
}
