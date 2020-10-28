| Task Summary |   |
| ----------------- | - |
| API Version | {{.APIVersion}}
| Kind    | {{.Kind}}
| Namespace   | {{.Namespace}}
| Name    | {{.Name}}
| Status  | {{ range .Status.Conditions }}{{.Reason}}{{end}} |
| Details | {{ range .Status.Conditions }}{{.Message}}{{end}} |
| Start   | {{ .Status.StartTime.UTC }} |
| End     | {{ .Status.CompletionTime.UTC }} |

## Steps

| Name | Status | Start | End
| ---- | ------ | ----- | ---
{{ range .Status.Steps -}}
| {{.Name}} |  {{.ContainerState.Terminated.Reason}} | {{.ContainerState.Terminated.StartedAt.UTC}} | {{.ContainerState.Terminated.FinishedAt.UTC}}
{{- end}}

```
{{yaml .Spec -}}
```