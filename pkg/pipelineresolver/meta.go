package pipelineresolver

import "net/http"

type Metadata struct {
	Header     http.Header
	Extensions map[string]interface{}
	Body       map[string]interface{}
	Params     map[string]interface{}
}
