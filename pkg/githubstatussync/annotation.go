package githubstatussync

import "fmt"

type annotationKey string

func (a annotationKey) String() string {
	return fmt.Sprintf("github.tekton.dev/%s", string(a))
}

const (
	ownerKey  = annotationKey("owner")
	repoKey   = annotationKey("repo")
	refKey    = annotationKey("ref")
	urlKey    = annotationKey("url")
	nameKey   = annotationKey("name")
	logServer = annotationKey("log-server")
)
