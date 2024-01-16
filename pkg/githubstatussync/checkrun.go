package githubstatussync

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/go-github/v43/github"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/logging"
)

func checkRunStepLog(
	tr *v1beta1.TaskRun,
	step v1beta1.StepState,
	url string,
	step_emoji string,
) string {
	return fmt.Sprintf(
		"Raw log for step: [%s](%s/%s/%s/%s) %s.",
		step.Name,
		tr.Annotations[logServer.String()],
		tr.Namespace,
		tr.Status.PodName,
		step.ContainerName,
		step_emoji,
	)
}

func checkRunStepAnnotation(step v1beta1.StepState, step_status string) *github.CheckRunAnnotation {
	return &github.CheckRunAnnotation{
		Path:      github.String("README.md"), // Dummy file name, required item.
		StartLine: github.Int(1),              // Dummy int, required item.
		EndLine:   github.Int(1),              // Dummy int, required item.
		AnnotationLevel: github.String(
			step_status,
		), // Can be one of notice, warning, or failure.
		Title: github.String(step.Name),
		Message: github.String(
			fmt.Sprintf("Task %s was finished, reason: %s.", step.Name, step.Terminated.Reason),
		),
		RawDetails: github.String(step.Terminated.Message),
	}
}

func checkRunOutput(tr *v1beta1.TaskRun, url string) *github.CheckRunOutput {
	var checkRunLogs []string
	var checkRunAnnotations []*github.CheckRunAnnotation

	for _, step := range tr.Status.Steps {
		var step_status, step_emoji string

		if step.Terminated == nil {
			fmt.Printf("TaskRun terminated field is nil, skip annotation. TR: %s \n", tr.Name)
			continue
		}

		switch step.Terminated.Reason {
		case "Completed":
			step_status = "notice"
			step_emoji = ":white_check_mark:"
		case "Failed", "Error":
			step_status = "failure"
			step_emoji = ":x:"
		case "TaskRunCancelled":
			step_status = "warning"
			step_emoji = ":warning:"
		default:
			step_status = "notice"
			step_emoji = ":grey_question:"
		}

		checkRunAnnotations = append(checkRunAnnotations, checkRunStepAnnotation(step, step_status))
		checkRunLogs = append(checkRunLogs, checkRunStepLog(tr, step, url, step_emoji))
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
		checkRunOptions.Conclusion = github.String(conclusion(ctx, eventType, tr))
	}

	logger.Warnf(
		"Trying to report %s status",
		eventType,
	)

	return checkRunOptions, nil
}
