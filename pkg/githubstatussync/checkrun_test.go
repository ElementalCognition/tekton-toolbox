package githubstatussync

import (
	"fmt"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/google/go-github/v43/github"
	v1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCheckRunOutput(t *testing.T) {
	newAnnotation := func(level, title, message, details string) *github.CheckRunAnnotation {
		return &github.CheckRunAnnotation{
			Path:            github.String("README.md"),
			StartLine:       github.Int(1),
			EndLine:         github.Int(1),
			AnnotationLevel: github.String(level),
			Title:           github.String(title),
			Message:         github.String(message),
			RawDetails:      github.String(details),
		}
	}
	newTaskRun := func(terminatedStates ...*corev1.ContainerStateTerminated) *v1beta1.TaskRun {
		steps := make([]v1beta1.StepState, len(terminatedStates))
		for i, state := range terminatedStates {
			steps[i] = v1beta1.StepState{
				ContainerState: corev1.ContainerState{Terminated: state},
				Name:           fmt.Sprintf("step%d", i+1),
				ContainerName:  fmt.Sprintf("step%d-container", i+1),
			}
		}
		return &v1beta1.TaskRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:        "test-taskrun",
				Annotations: map[string]string{logServer.String(): "https://log.example.com"},
				Namespace:   "default",
			},
			Status: v1beta1.TaskRunStatus{
				TaskRunStatusFields: v1beta1.TaskRunStatusFields{
					PodName: "test-pod",
					Steps:   steps,
				},
			},
		}
	}
	verifyOutput := func(t *testing.T, output, expectedOutput *github.CheckRunOutput) {
		t.Helper()
		ignoreUnexported := cmpopts.IgnoreUnexported(github.CheckRunOutput{})
		if !cmp.Equal(output, expectedOutput, ignoreUnexported) {
			t.Errorf("checkRunOutput() mismatch (-want +got):\n%s", cmp.Diff(expectedOutput, output, ignoreUnexported))
		}
	}
	tr := newTaskRun(
		&corev1.ContainerStateTerminated{Reason: "Completed", Message: "Step completed successfully"},
		&corev1.ContainerStateTerminated{Reason: "Failed", Message: "Step failed"},
	)
	sampleURL := "https://tekton.example.com"
	output := checkRunOutput(tr, sampleURL)
	expectedAnnotations := []*github.CheckRunAnnotation{
		newAnnotation("notice", "step1", "Task step1 was finished, reason: Completed.", "Step completed successfully"),
		newAnnotation("failure", "step2", "Task step2 was finished, reason: Failed.", "Step failed"),
	}
	expectedOutput := &github.CheckRunOutput{
		Title:   github.String("Steps details"),
		Summary: github.String(fmt.Sprintf("You can find more details on %s. Check the raw logs if data is no longer available on Tekton Dashboard.", sampleURL)),
		Text: github.String(strings.Join([]string{"Raw log for step: [step1](https://log.example.com/default/test-pod/step1-container) :white_check_mark:.",
			"Raw log for step: [step2](https://log.example.com/default/test-pod/step2-container) :x:."}, "</br>")),
		Annotations: expectedAnnotations,
	}
	verifyOutput(t, output, expectedOutput)
}
