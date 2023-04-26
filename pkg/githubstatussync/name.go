package githubstatussync

import (
	"bytes"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"html/template"
)

const defaultName = "{{ .Namespace }}/{{ .Name }}"

func nameFor(tr *v1beta1.TaskRun) (string, error) {
	name, ok := tr.Annotations[nameKey.String()]
	if !ok || len(name) == 0 {
		name = defaultName
	}
	t, err := template.New("name").Parse(name)
	if err != nil {
		return "", err
	}
	var tpl bytes.Buffer
	if err := t.Execute(&tpl, tr); err != nil {
		return "", err
	}
	return tpl.String(), nil
}
