package pipelineresolver

import (
	"context"
	"errors"
)

var (
	contextKey  = struct{}{}
	errNotFound = errors.New("resolver does not exist")
)

func WithResolver(ctx context.Context, r Resolver) context.Context {
	return context.WithValue(ctx, contextKey, r)
}

func FromContext(ctx context.Context) (Resolver, error) {
	val, ok := ctx.Value(contextKey).(Resolver)
	if !ok {
		return nil, errNotFound
	}
	return val, nil
}
