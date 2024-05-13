package githubstatussync

import (
	"bytes"
	v1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"html/template"
)

const defaultDetailsURL = "https://tekton.dev/#/namespaces/{{ .Namespace }}/taskruns/{{ .Name }}"

func detailsURL(tr *v1.TaskRun) (string, error) {
	url, ok := tr.Annotations[urlKey.String()]
	if !ok || len(url) == 0 {
		url = defaultDetailsURL
	}
	t, err := template.New("url").Parse(url)
	if err != nil {
		return "", err
	}
	var tpl bytes.Buffer
	if err = t.Execute(&tpl, tr); err != nil {
		return "", err
	}
	return tpl.String(), nil
}
