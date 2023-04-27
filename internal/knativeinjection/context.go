package knativeinjection

import (
	"context"
	"k8s.io/client-go/rest"
	"knative.dev/pkg/injection"
)

func EnableInjectionOrDie(cfg *rest.Config) context.Context {
	ctx, _ := injection.EnableInjectionOrDie(context.Background(), cfg)
	return ctx
}
