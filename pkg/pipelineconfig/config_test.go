package pipelineconfig

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfig_UnmarshalYAML(t *testing.T) {
	buf, err := os.ReadFile("testdata/config.yaml")
	assert.Nil(t, err)
	var cfg Config
	err = cfg.UnmarshalYAML(buf)
	assert.Nil(t, err)
	assert.NotNil(t, cfg.Defaults)
	assert.NotNil(t, cfg.Triggers)
	assert.Len(t, cfg.Defaults.Params, 4)
	assert.Len(t, cfg.Triggers, 2)
}

func TestConfig_UnmarshalYAML_MarshalJSON(t *testing.T) {
	buf, err := os.ReadFile("testdata/config.yaml")
	assert.Nil(t, err)
	var cfg Config
	err = cfg.UnmarshalYAML(buf)
	assert.Nil(t, err)
	b, err := cfg.MarshalJSON()
	assert.Nil(t, err)
	assert.NotNil(t, b)
}
