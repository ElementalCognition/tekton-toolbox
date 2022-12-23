package githubstatussync

import (
	t "time"

	"github.com/google/go-github/v48/github"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func timestamp(t *metav1.Time) *github.Timestamp {
	if t == nil {
		return nil
	}
	return &github.Timestamp{Time: t.Time}
}

func time(t *github.Timestamp) *t.Time {
	if t == nil {
		return nil
	}
	return &t.Time
}
