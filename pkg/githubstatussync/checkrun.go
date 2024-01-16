package githubstatussync

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v43/github"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/logging"
)

func checkRunOutput(tr *v1beta1.TaskRun, url string) *github.CheckRunOutput {
	var chkRunAnno []*github.CheckRunAnnotation
	var logs []string
	for _, v := range tr.Status.Steps {
		var s, e string
		if v.Terminated == nil {
			fmt.Printf("TaskRun terminated field is nil, skip annotation. TR: %s \n", tr.Name)
			continue
		}
		switch v.Terminated.Reason {
		case "Completed":
			s = "notice"
			e = ":white_check_mark:"
		case "Failed", "Error":
			s = "failure"
			e = ":x:"
		case "TaskRunCancelled":
			s = "warning"
			e = ":warning:"
		default:
			s = "notice"
			e = ":grey_question:"
		}
		logs = append(
			logs,
			fmt.Sprintf(
				"Raw log for step: [%s](%s/%s/%s/%s) %s.",
				v.Name,
				tr.Annotations[logServer.String()],
				tr.Namespace,
				tr.Status.PodName,
				v.ContainerName,
				e,
			),
		)
		chkRunAnno = append(chkRunAnno, &github.CheckRunAnnotation{
			Path:      github.String("README.md"), // Dummy file name, required item.
			StartLine: github.Int(1),              // Dummy int, required item.
			EndLine:   github.Int(1),              // Dummy int, required item.
			AnnotationLevel: github.String(
				s,
			), // Can be one of notice, warning, or failure.
			Title: github.String(v.Name),
			Message: github.String(
				fmt.Sprintf("Task %s was finished, reason: %s.", v.Name, v.Terminated.Reason),
			),
			RawDetails: github.String(v.Terminated.Message),
		})
	}

	return &github.CheckRunOutput{
		Title: github.String("Steps details"),
		Summary: github.String(
			fmt.Sprintf(
				"You can find more details on %s. Check the raw logs if data is no longer available on Tekton Dashboard.",
				url,
			),
		),
		Text:        github.String(strings.Join(logs, "</br>")),
		Annotations: chkRunAnno,
	}
}

func checkRun(
	ctx context.Context,
	eventType string,
	tr *v1beta1.TaskRun,
) (*github.CreateCheckRunOptions, error) {
	logger := logging.FromContext(ctx)

	url, err := detailsURL(tr)
	if err != nil {
		return nil, err
	}
	name, err := nameFor(tr)
	if err != nil {
		return nil, err
	}
	ref := tr.Annotations[refKey.String()]
	output := checkRunOutput(tr, url)
	status := status(eventType)

	checkRunOptions := &github.CreateCheckRunOptions{
		ExternalID: github.String(string(tr.UID)),
		Name:       name,
		Status:     github.String(status),
		HeadSHA:    ref,
		StartedAt:  timestamp(tr.Status.StartTime),
		DetailsURL: github.String(url),
		Output:     output,
	}

	// According to docs, conclusion is only available if the status is "completed".
	// Also, if you specify completedAt field, conclusion becomes required.
	// https://docs.github.com/en/rest/checks/runs?apiVersion=2022-11-28#create-a-check-run
	if status == checkRunStatusCompleted {
		logger.Warnf(
			"Found completed check run, adding conclusion and completedAt fields to the request",
		)
		checkRunOptions.CompletedAt = timestamp(tr.Status.CompletionTime)
		checkRunOptions.Conclusion = github.String(conclusion(ctx, status, tr))
	}

	logger.Warnf(
		"Trying to report %s status",
		eventType,
	)

	return checkRunOptions, nil
}
