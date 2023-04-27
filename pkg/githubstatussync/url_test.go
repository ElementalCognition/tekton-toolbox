package githubstatussync

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestDetailsURL(t *testing.T) {
	testCases := []struct {
		name        string
		taskRun     *v1beta1.TaskRun
		expectedURL string
	}{
		{
			name: "default URL",
			taskRun: &v1beta1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-taskrun",
					Namespace: "default",
				},
			},
			expectedURL: "https://tekton.dev/#/namespaces/default/taskruns/test-taskrun",
		},
		{
			name: "custom URL",
			taskRun: &v1beta1.TaskRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-taskrun",
					Namespace: "default",
					Annotations: map[string]string{
						"github.tekton.dev/url": "https://custom.example.com/{{ .Namespace }}/taskruns/{{ .Name }}",
					},
				},
			},
			expectedURL: "https://custom.example.com/default/taskruns/test-taskrun",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			url, err := detailsURL(tc.taskRun)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if url != tc.expectedURL {
				t.Errorf("detailsURL() mismatch (-want +got):\n%s", cmp.Diff(tc.expectedURL, url))
			}
		})
	}
}
