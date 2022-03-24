package pipelineresolver

import (
	"context"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestFromContext_Error(t *testing.T) {
	r, err := FromContext(context.TODO())
	assert.Nil(t, r)
	assert.NotNil(t, err)
}
