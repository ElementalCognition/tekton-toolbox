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
	// Set up test data
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
	tr := &v1beta1.TaskRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-taskrun",
			Annotations: map[string]string{
				logServer.String(): "https://log.example.com",
			},
			Namespace: "default",
		},
		Status: v1beta1.TaskRunStatus{
			TaskRunStatusFields: v1beta1.TaskRunStatusFields{
				PodName: "test-pod",
				Steps: []v1beta1.StepState{
					{
						ContainerState: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								Reason:  "Completed",
								Message: "Step completed successfully",
							},
						},
						Name:          "step1",
						ContainerName: "step1-clone",
					},
					{
						ContainerState: corev1.ContainerState{
							Terminated: &corev1.ContainerStateTerminated{
								Reason:  "Failed",
								Message: "Step failed",
							},
						},
						Name:          "step2",
						ContainerName: "step2-build",
					},
				},
			},
		},
	}

	sampleURL := "https://tekton.example.com"

	// Call the function
	output := checkRunOutput(tr, sampleURL)
	if len(output.Annotations) != 2 {
		t.Errorf("Expected 2 annotations, got %d", len(output.Annotations))
	}
	// Verify the output
	expectedAnnotations := []*github.CheckRunAnnotation{
		newAnnotation("notice", "step1", "Task step1 was finished, reason: Completed.", "Step completed successfully"),
		newAnnotation("failure", "step2", "Task step2 was finished, reason: Failed.", "Step failed"),
	}

	expectedOutput := &github.CheckRunOutput{
		Title:   github.String("Steps details"),
		Summary: github.String(fmt.Sprintf("You can find more details on %s. Check the raw logs if data is no longer available on Tekton Dashboard.", sampleURL)),
		Text: github.String(strings.Join([]string{"Raw log for step: [step1](https://log.example.com/default/test-pod/step1-clone) :white_check_mark:.",
			"Raw log for step: [step2](https://log.example.com/default/test-pod/step2-build) :x:."}, "</br>")),
		Annotations: expectedAnnotations,
	}
	ignoreUnexported := cmpopts.IgnoreUnexported(github.CheckRunOutput{})
	if !cmp.Equal(output, expectedOutput, ignoreUnexported) {
		t.Errorf("checkRunOutput() mismatch (-want +got):\n%s", cmp.Diff(expectedOutput, output, ignoreUnexported))
	}
}
