package pipelineresolver

import (
	"context"
)

type Resolver interface {
	ValueOf(ctx context.Context, meta *Metadata, val string) (interface{}, error)
	SafeValueOf(ctx context.Context, meta *Metadata, val string) (interface{}, error)
}
