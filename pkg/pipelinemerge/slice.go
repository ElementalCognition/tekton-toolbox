package pipelinemerge

import (
	"github.com/imdario/mergo"
	"reflect"
)

type sliceMergeTransformer struct {
	key  string
	opts []func(*mergo.Config)
}

var _ mergo.Transformers = (*sliceMergeTransformer)(nil)

func (t *sliceMergeTransformer) nameFor(v reflect.Value) (string, bool) {
	i := v.FieldByName(t.key).Interface()
	s, ok := i.(string)
	return s, ok
}

func (t *sliceMergeTransformer) refsFor(v reflect.Value) map[string]int {
	refs := map[string]int{}
	for i := 0; i < v.Len(); i++ {
		el := v.Index(i)
		if el.Kind() == reflect.Struct {
			if s, ok := t.nameFor(el); ok {
				refs[s] = i
			}
		}
	}
	return refs
}

func (t *sliceMergeTransformer) merge(dst reflect.Value, src reflect.Value) error {
	dstVal := dst.Addr().Interface()
	if src.CanInterface() {
		src = reflect.ValueOf(src.Interface())
	}
	srcVal := src.Interface()
	if err := mergo.Merge(dstVal, srcVal, t.opts...); err != nil {
		return err
	}
	return nil
}

func (t *sliceMergeTransformer) mergeRefs(i int, refs map[string]int, dst reflect.Value, src reflect.Value) error {
	srcEl := src.Index(i)
	switch srcEl.Kind() {
	case reflect.Struct:
		if s, ok := t.nameFor(srcEl); ok {
			if dstIndex, ok := refs[s]; ok {
				dstEl := dst.Index(dstIndex)
				return t.merge(dstEl, srcEl)
			}
			refs[s] = i
			dst.Set(reflect.Append(dst, srcEl))
		} else {
			dst.Set(reflect.Append(dst, srcEl))
		}
	default:
		dst.Set(reflect.Append(dst, srcEl))
	}
	return nil
}

func (t *sliceMergeTransformer) Transformer(typ reflect.Type) func(dst reflect.Value, src reflect.Value) error {
	if typ.Kind() != reflect.Slice {
		return nil
	}
	return func(dst reflect.Value, src reflect.Value) error {
		if !dst.CanSet() {
			return nil
		}
		if !dst.CanAddr() {
			return mergo.ErrNonPointerAgument
		}
		dstRefs := t.refsFor(dst)
		for i := 0; i < src.Len(); i++ {
			if err := t.mergeRefs(i, dstRefs, dst, src); err != nil {
				return err
			}
		}
		return nil
	}
}

func WithMergeSliceByKey(key string, opts ...func(*mergo.Config)) func(cfg *mergo.Config) {
	return func(cfg *mergo.Config) {
		cfg.Transformers = &sliceMergeTransformer{
			key:  key,
			opts: opts,
		}
	}
}

func WithMergeSliceByName(opts ...func(*mergo.Config)) func(cfg *mergo.Config) {
	return WithMergeSliceByKey("Name", opts...)
}
