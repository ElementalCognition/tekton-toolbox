package githubstatussync

import (
	"fmt"
	"strings"

	"github.com/google/go-github/v43/github"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

func checkRunOutput(tr *v1beta1.TaskRun, url string) *github.CheckRunOutput {
	var chkRunAnno []*github.CheckRunAnnotation
	var logs []string
	for _, step := range tr.Status.Steps {
		if step.Terminated == nil {
			fmt.Printf("TaskRun terminated field is nil, skip annotation. TR: %s \n", tr.Name)
			continue
		}
		terminationReason := step.Terminated.Reason
		annotationLevel, emoji := determineAnnotationLevelAndEmoji(terminationReason)

		logs = append(logs, fmt.Sprintf("Raw log for step: [%s](%s/%s/%s/%s) %s.", step.Name, tr.Annotations[logServer.String()], tr.Namespace, tr.Status.PodName, step.ContainerName, emoji))
		chkRunAnno = append(chkRunAnno, &github.CheckRunAnnotation{
			Path:            github.String("README.md"),     // Dummy file name, required item.
			StartLine:       github.Int(1),                  // Dummy int, required item.
			EndLine:         github.Int(1),                  // Dummy int, required item.
			AnnotationLevel: github.String(annotationLevel), // Can be one of notice, warning, or failure.
			Title:           github.String(step.Name),
			Message:         github.String(fmt.Sprintf("Task %s was finished, reason: %s.", step.Name, terminationReason)),
			RawDetails:      github.String(step.Terminated.Message),
		})
	}

	return &github.CheckRunOutput{
		Title:       github.String("Steps details"),
		Summary:     github.String(fmt.Sprintf("You can find more details on %s. Check the raw logs if data is no longer available on Tekton Dashboard.", url)),
		Text:        github.String(strings.Join(logs, "</br>")),
		Annotations: chkRunAnno,
	}
}

func determineAnnotationLevelAndEmoji(reason string) (string, string) {
	switch reason {
	case "Completed":
		return "notice", ":white_check_mark:"
	case "Failed", "Error":
		return "failure", ":x:"
	case "TaskRunCancelled":
		return "warning", ":warning:"
	default:
		return "notice", ":grey_question:"
	}
}

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
	output := checkRunOutput(tr, url)
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
