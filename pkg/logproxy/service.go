package logproxy

import (
	"context"
)

type Service interface {
	Fetch(ctx context.Context, namespace, pod, container string) ([]byte, error)
}
