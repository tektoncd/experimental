package pipelinerun

import (
	"bytes"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"text/template"
)

type templateVariables struct {
	Namespace string
	Name      string
	Version   string
	Kind      string
	Group     string
}

func dashboardURL(pr *v1beta1.PipelineRun) string {
	url, ok := pr.Annotations[statusTargetURLName]

	if !ok || len(url) == 0 {
		return ""
	}

	t, err := template.New("dashboardURL").Parse(url)

	if err != nil {
		return ""
	}
	var tpl bytes.Buffer
	variables := templateVariables{
		pr.Namespace,
		pr.Name,
		pr.GroupVersionKind().Version,
		pr.GroupVersionKind().Kind,
		pr.GroupVersionKind().Group,
	}

	if err := t.Execute(&tpl, variables); err != nil {
		return ""
	}

	return tpl.String()
}
