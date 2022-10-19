package parse

import (
	"fmt"
	"testing"

	fluxnotifications "github.com/fluxcd/notification-controller/api/v1beta1"
	fluxsource "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func mustParseYAML(t *testing.T, yaml string, i runtime.Object) {
	if _, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(yaml), nil, i); err != nil {
		t.Fatalf("mustParseYAML (%s): %v", yaml, err)
	}
}

// MustParsePipelineRun takes YAML and parses it into a *v1beta1.PipelineRun
func MustParsePipelineRun(t *testing.T, yaml string) *pipelinev1beta1.PipelineRun {
	var pr pipelinev1beta1.PipelineRun
	yaml = `apiVersion: tekton.dev/v1beta1
kind: PipelineRun
` + yaml
	mustParseYAML(t, yaml, &pr)
	return &pr
}

// MustParsePipelineRun takes YAML and parses it into a *v1beta1.PipelineRun
func MustParseWorkflow(t *testing.T, name, namespace, yaml string) *v1alpha1.Workflow {
	var w v1alpha1.Workflow
	yaml = fmt.Sprintf(`apiVersion: tekton.dev/v1alpha1
kind: Workflow
metadata:
  name: %s
  namespace: %s
`+yaml, name, namespace)
	mustParseYAML(t, yaml, &w)
	return &w
}

// MustParseTriggerTemplate takes YAML and parses it into a *triggersv1beta1.TriggerTemplate
func MustParseTriggerTemplate(t *testing.T, yaml string) *triggersv1beta1.TriggerTemplate {
	var tt triggersv1beta1.TriggerTemplate
	yaml = `apiVersion: triggers.tekton.dev/v1beta1
kind: TriggerTemplate
` + yaml
	mustParseYAML(t, yaml, &tt)
	return &tt
}

// MustParseTrigger takes YAML and parses it into a *triggersv1beta1.Trigger
func MustParseTrigger(t *testing.T, yaml string) *triggersv1beta1.Trigger {
	var tr triggersv1beta1.Trigger
	yaml = `apiVersion: triggers.tekton.dev/v1beta1
kind: Trigger
` + yaml
	mustParseYAML(t, yaml, &tr)
	return &tr
}

func MustParseRepo(t *testing.T, name, namespace, yaml string) fluxsource.GitRepository {
	var repo fluxsource.GitRepository
	yaml = fmt.Sprintf(`
kind: GitRepository
apiVersion: source.toolkit.fluxcd.io/v1beta2
metadata:
  name: %s
  namespace: %s
`+yaml, name, namespace)
	mustParseYAML(t, yaml, &repo)
	return repo
}

func MustParseReceiver(t *testing.T, name, namespace, yaml string) fluxnotifications.Receiver {
	var r fluxnotifications.Receiver
	yaml = fmt.Sprintf(`
kind: Receiver
apiVersion: notifications.toolkit.fluxcd.io/v1beta1
metadata:
  name: %s
  namespace: %s
`+yaml, name, namespace)
	mustParseYAML(t, yaml, &r)
	return r
}

func MustParseProvider(t *testing.T, name, namespace, yaml string) fluxnotifications.Provider {
	var p fluxnotifications.Provider
	yaml = fmt.Sprintf(`
apiVersion: notifications.toolkit.fluxcd.io/v1beta1
kind: Provider
metadata:
  name: %s
  namespace: %s
`+yaml, name, namespace)
	mustParseYAML(t, yaml, &p)
	return p
}

func MustParseAlert(t *testing.T, name, namespace, yaml string) fluxnotifications.Alert {
	var a fluxnotifications.Alert
	yaml = fmt.Sprintf(`
apiVersion: notifications.toolkit.fluxcd.io/v1beta1
kind: Alert
metadata:
  name: %s
  namespace: %s
`+yaml, name, namespace)
	mustParseYAML(t, yaml, &a)
	return a
}
