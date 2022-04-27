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
	for _, v := range tr.Status.Steps {
		var s, e string
		switch v.Terminated.Reason {
		case "Completed":
			s = "notice"
			e = ":white_check_mark:"
		case "Failed":
			s = "failure"
			e = ":x:"
		case "Error":
			s = "failure"
			e = ":x:"
		case "TaskRunCancelled":
			s = "warning"
			e = ":warning:"
		default:
			s = "notice"
			e = ":grey_question:"
		}
		logs = append(logs, fmt.Sprintf("- Raw log for step[%s](%s/%s/%s/%s) %s .  ", v.Name, tr.Annotations[logServer.String()], tr.Namespace, tr.Status.PodName, v.ContainerName, e))
		chkRunAnno = append(chkRunAnno, &github.CheckRunAnnotation{
			Path:            github.String("README.md"), // Dummy file name, required item.
			StartLine:       github.Int(1),              // Dummy int, required item.
			EndLine:         github.Int(1),              // Dummy int, required item.
			AnnotationLevel: github.String(s),           // Can be one of notice, warning, or failure.
			Title:           github.String(v.Name),
			Message:         github.String(fmt.Sprintf("Task %s was finished, reason: %s.", v.Name, v.Terminated.Reason)),
			RawDetails:      github.String(v.Terminated.Message),
		})
	}
	return &github.CheckRunOutput{
		Title:       github.String("Steps details"),
		Summary:     github.String(fmt.Sprintf("You can find more details on %s. Check the raw logs if data is no longer available on %s.", url, url)),
		Text:        github.String(strings.Join(logs, "")),
		Annotations: chkRunAnno,
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
