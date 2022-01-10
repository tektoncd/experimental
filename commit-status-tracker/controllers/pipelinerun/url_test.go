package pipelinerun

import (
	"testing"

	"k8s.io/apimachinery/pkg/util/diff"
)

func TestDashboardURL(t *testing.T) {
	for _, tc := range []struct {
		detailsURLAnnotation string
		wantDetailsURL       string
	}{
		{
			detailsURLAnnotation: "https://tekton.dev",
			wantDetailsURL:       "https://tekton.dev",
		},
		{
			detailsURLAnnotation: "https://dashboard.dogfooding.tekton.dev/#/namespaces/{{ .Namespace }}/pipelineruns/{{ .Name }}",
			wantDetailsURL:       "https://dashboard.dogfooding.tekton.dev/#/namespaces/test-namespace/pipelineruns/test-pipeline-run",
		},
		{
			detailsURLAnnotation: "https://console-openshift-console.apps-crc.testing/k8s/ns/{{ .Namespace }}/{{ .Group }}~{{ .Version }}~{{ .Kind }}/{{ .Name }}",
			wantDetailsURL:       "https://console-openshift-console.apps-crc.testing/k8s/ns/test-namespace/tekton.dev~v1beta1~PipelineRun/test-pipeline-run",
		},
	} {
		t.Run(tc.detailsURLAnnotation, func(t *testing.T) {
			pr := makePipelineRunWithResources()
			pr.Annotations = make(map[string]string)
			pr.Annotations[statusTargetURLName] = tc.detailsURLAnnotation

			url := dashboardURL(pr)

			if tc.wantDetailsURL != url {
				t.Errorf("-want,+got:\n%s", diff.StringDiff(tc.wantDetailsURL, url))
			}

		})
	}
}
