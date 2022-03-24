package pipelineresolver

import (
	"context"
)

type Reconciler interface {
	Reconcile(ctx context.Context, meta *Metadata) error
}
