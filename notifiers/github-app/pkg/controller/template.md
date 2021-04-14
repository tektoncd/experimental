| Task Summary |   |
| ----------------- | - |
| API Version | {{.APIVersion}}
| Kind    | {{.Kind}}
| Namespace   | {{.Namespace}}
| Name    | {{.Name}}
| Status  | {{ range .Status.Conditions }}{{.Reason}}{{end}} |
| Details | {{ range .Status.Conditions }}{{.Message}}{{end}} |
| Start   | {{ .Status.StartTime.UTC }} |
| End     | {{ with .Status.CompletionTime -}} {{ .UTC }} {{- end }} |

## Steps

| Name | Status | Start | End |
| ---- | ------ | ----- | --- |
{{ range .Status.Steps -}}
| {{.Name}} | {{ with .ContainerState.Terminated -}} {{.Reason}} | {{.StartedAt.UTC}} | {{.FinishedAt.UTC}} {{- else }} | | {{- end }} |
{{- end}}

```
{{yaml .Spec -}}
```