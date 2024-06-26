package githubstatussync

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v43/github"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"knative.dev/pkg/logging"
)

func checkRunStepLogString(tr *v1.TaskRun, step v1.StepState, emoji string) string {
	return fmt.Sprintf(
		"Raw log for step: [%s](%s/%s/%s/%s) %s.",
		step.Name,
		tr.Annotations[logServer.String()],
		tr.Namespace,
		tr.Status.PodName,
		step.Container,
		emoji,
	)
}

func checkRunStepAnnotation(step v1.StepState, status string) *github.CheckRunAnnotation {
	return &github.CheckRunAnnotation{
		Path:      github.String("README.md"), // Dummy file name, required item.
		StartLine: github.Int(1),              // Dummy int, required item.
		EndLine:   github.Int(1),              // Dummy int, required item.
		AnnotationLevel: github.String(
			status,
		), // Can be one of notice, warning, or failure.
		Title: github.String(step.Name),
		Message: github.String(
			fmt.Sprintf("Task %s was finished, reason: %s.", step.Name, step.Terminated.Reason),
		),
		RawDetails: github.String(step.Terminated.Message),
	}
}

func checkRunOutput(ctx context.Context, tr *v1.TaskRun, url string) *github.CheckRunOutput {
	logger := logging.FromContext(ctx)
	var checkRunLogs []string
	var checkRunAnnotations []*github.CheckRunAnnotation

	for _, step := range tr.Status.Steps {
		var stepStatus, stepEmoji string

		if step.Terminated == nil {
			logger.Debugw("TaskRun terminated field is nil, skip annotation. TR: %s \n", tr.Name)
			continue
		}

		switch step.Terminated.Reason {
		case "Completed":
			stepStatus = "notice"
			stepEmoji = ":white_check_mark:"
		case "Failed", "Error":
			stepStatus = "failure"
			stepEmoji = ":x:"
		case "TaskRunCancelled":
			stepStatus = "warning"
			stepEmoji = ":warning:"
		default:
			stepStatus = "notice"
			stepEmoji = ":grey_question:"
		}

		checkRunAnnotations = append(checkRunAnnotations, checkRunStepAnnotation(step, stepStatus))
		checkRunLogs = append(checkRunLogs, checkRunStepLogString(tr, step, stepEmoji))
	}

	return &github.CheckRunOutput{
		Title: github.String("Steps details"),
		Summary: github.String(
			fmt.Sprintf(
				"You can find more details on %s. Check the raw logs if data is no longer available on Tekton Dashboard.",
				url,
			),
		),
		Text:        github.String(strings.Join(checkRunLogs, "</br>")),
		Annotations: checkRunAnnotations,
	}
}

func checkRun(
	ctx context.Context,
	eventType string,
	tr *v1.TaskRun,
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
	output := checkRunOutput(ctx, tr, url)
	status := getStatus(eventType)

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
		logger.Debugw(
			"Found completed check run, adding conclusion and completedAt fields to the request",
		)
		checkRunOptions.CompletedAt = timestamp(tr.Status.CompletionTime)
		checkRunOptions.Conclusion = github.String(resolveConclusion(ctx, eventType, tr))
	}

	logger.Debugw(
		"Trying to report %s status",
		eventType,
	)

	return checkRunOptions, nil
}
