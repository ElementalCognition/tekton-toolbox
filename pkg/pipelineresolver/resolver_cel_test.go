package pipelineresolver

import (
	"context"
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestNewCelResolverEnv(t *testing.T) {
	env, err := NewCelEnv()
	assert.Nil(t, err)
	assert.NotNil(t, env)
}

func TestNewCelResolver(t *testing.T) {
	r, err := NewCelResolver()
	assert.Nil(t, err)
	assert.NotNil(t, r)
}

func TestCelResolver_ValueOf_Body(t *testing.T) {
	req := &Metadata{
		Body: map[string]interface{}{
			"repository": map[string]interface{}{
				"url": "foo",
			},
		},
	}
	r, err := NewCelResolver()
	assert.Nil(t, err)
	val, err := r.ValueOf(context.TODO(), req, "body.repository.url")
	assert.Nil(t, err)
	assert.Equal(t, "foo", val)
}

func TestCelResolver_ValueOf_Header(t *testing.T) {
	header := http.Header{}
	header.Add("foo", "bar")
	req := &Metadata{
		Header: header,
	}
	r, err := NewCelResolver()
	assert.Nil(t, err)
	val, err := r.ValueOf(context.TODO(), req, "header.Foo[0]")
	assert.Nil(t, err)
	assert.Equal(t, "bar", val)
}

func TestCelResolver_ValueOf_Extensions(t *testing.T) {
	req := &Metadata{
		Extensions: map[string]interface{}{
			"foo": "bar",
		},
	}
	r, err := NewCelResolver()
	assert.Nil(t, err)
	val, err := r.ValueOf(context.TODO(), req, "extensions.foo")
	assert.Nil(t, err)
	assert.Equal(t, "bar", val)
}
