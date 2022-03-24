package pipelineresolver

import (
	"context"
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	celext "github.com/google/cel-go/ext"
	triggerscel "github.com/tektoncd/triggers/pkg/interceptors/cel"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func NewCelEnv() (*cel.Env, error) {
	mapStrDyn := decls.NewMapType(decls.String, decls.Dyn)
	return cel.NewEnv(
		triggerscel.Triggers(metav1.NamespaceAll, nil),
		celext.Strings(),
		celext.Encoders(),
		cel.Declarations(
			decls.NewVar("body", mapStrDyn),
			decls.NewVar("header", mapStrDyn),
			decls.NewVar("extensions", mapStrDyn),
			decls.NewVar("requestURL", decls.String),
			decls.NewVar("params", mapStrDyn),
		))
}

type CelResolver struct {
	env *cel.Env
}

var _ Resolver = (*CelResolver)(nil)

func (r *CelResolver) ValueOf(_ context.Context, meta *Metadata, val string) (interface{}, error) {
	ast, issues := r.env.Parse(val)
	if issues != nil && issues.Err() != nil {
		return nil, &ExprInvalidError{issues.Err()}
	}
	ast, issues = r.env.Check(ast)
	if issues != nil && issues.Err() != nil {
		return nil, &ExprInvalidError{issues.Err()}
	}
	prg, err := r.env.Program(ast)
	if err != nil {
		return nil, err
	}
	i, _, err := prg.Eval(map[string]interface{}{
		"body":       meta.Body,
		"header":     meta.Header,
		"extensions": meta.Extensions,
		"params":     meta.Params,
	})
	if err != nil {
		return nil, err
	}
	return i.Value(), nil
}

func (r *CelResolver) SafeValueOf(ctx context.Context, meta *Metadata, val string) (interface{}, error) {
	i, err := r.ValueOf(ctx, meta, val)
	if err != nil {
		i = val
		switch err.(type) {
		case *ExprInvalidError:
		default:
			return nil, err
		}
	}
	return i, nil
}

func NewCelResolver() (Resolver, error) {
	env, err := NewCelEnv()
	if err != nil {
		return nil, err
	}
	return &CelResolver{
		env: env,
	}, nil
}
