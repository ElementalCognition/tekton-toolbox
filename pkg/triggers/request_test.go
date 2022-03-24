package triggers

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestInterceptorRequest_UnmarshalBody(t *testing.T) {
	buf, err := os.ReadFile("testdata/payload.json")
	assert.Nil(t, err)
	var tr InterceptorRequest
	err = json.Unmarshal(buf, &tr)
	assert.Nil(t, err)
	body, err := tr.UnmarshalBody()
	assert.Nil(t, err)
	assert.Equal(t, map[string]interface{}{
		"foo": "bar",
	}, body)
}
