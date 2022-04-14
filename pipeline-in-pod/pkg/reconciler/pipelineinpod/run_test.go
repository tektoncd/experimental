package pipelineinpod_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	cprv1alpha1 "github.com/tektoncd/experimental/pipeline-in-pod/pkg/apis/colocatedpipelinerun/v1alpha1"
	"github.com/tektoncd/experimental/pipeline-in-pod/pkg/reconciler/pipelineinpod"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test/parse"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
)

func TestToColocatedPipelineRun(t *testing.T) {
	cprSpec := cprv1alpha1.ColocatedPipelineRunSpec{
		Timeouts: &v1beta1.TimeoutFields{Pipeline: &metav1.Duration{Duration: time.Duration(10 * time.Second)}},
	}
	specBytes, err := json.Marshal(cprSpec)
	if err != nil {
		t.Fatalf("%s", err)
	}
	cprStatus := cprv1alpha1.ColocatedPipelineRunStatus{}
	statusBytes, err := json.Marshal(cprStatus)
	if err != nil {
		t.Fatalf("%s", err)
	}
	run := v1alpha1.Run{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Run",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-run",
			Namespace: "default",
		},
		Spec: v1alpha1.RunSpec{
			Spec: &v1alpha1.EmbeddedRunSpec{
				TypeMeta: runtime.TypeMeta{
					Kind:       "ColocatedPipelineRun",
					APIVersion: "tekton.dev/v1alpha1",
				},
				Metadata: v1beta1.PipelineTaskMetadata{
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: runtime.RawExtension{
					Raw: specBytes,
				},
			},
		},
		Status: v1alpha1.RunStatus{
			RunStatusFields: v1alpha1.RunStatusFields{
				ExtraFields: runtime.RawExtension{
					Raw: statusBytes,
				},
			},
		},
	}
	cpr, err := pipelineinpod.ToColocatedPipelineRun(&run)
	expected := cprv1alpha1.ColocatedPipelineRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ColocatedPipelineRun",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"foo": "bar",
			},
			Namespace: "default",
			Name:      "my-run",
		},
		Spec:   cprSpec,
		Status: cprStatus,
	}
	if err != nil {
		t.Errorf("got err %s", err)
	}
	if diff := cmp.Diff(cpr, expected); diff != "" {
		t.Errorf("got != want: %s", diff)
	}
}

func TestToColocatedPipelineRunFromYaml(t *testing.T) {
	run := parse.MustParseRun(t, `
apiVersion: tekton.dev/v1alpha1
kind: Run
metadata:
  name: echo-good-morning-run
spec:
  spec:
    apiVersion: tekton.dev/v1alpha1
    kind: ColocatedPipelineRun
    spec:
      params:
      - name: greeting
        value: "Good Morning!"
      timeouts:
        pipeline: 15s
      pipelineSpec:
        params:
        - name: greeting
          type: string
        tasks:
        - name: echo-good-morning
          params:
          - name: greeting
            value: $(params.greeting)
          taskSpec:
            steps:
            - name: echo
              image: ubuntu
`)
	cpr, err := pipelineinpod.ToColocatedPipelineRun(run)
	if err != nil {
		t.Fatalf("error parsing run yaml: %s", err)
	}
	expected := cprv1alpha1.ColocatedPipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Name: "echo-good-morning-run",
		},
		TypeMeta: metav1.TypeMeta{Kind: "ColocatedPipelineRun", APIVersion: "tekton.dev/v1alpha1"},
		Spec: cprv1alpha1.ColocatedPipelineRunSpec{
			Params: []v1beta1.Param{
				{Name: "greeting", Value: v1beta1.ArrayOrString{StringVal: "Good Morning!", Type: v1beta1.ParamTypeString}},
			},
			PipelineSpec: &v1beta1.PipelineSpec{
				Params: []v1beta1.ParamSpec{
					{Name: "greeting", Type: v1beta1.ParamTypeString},
				},
				Tasks: []v1beta1.PipelineTask{{
					Name: "echo-good-morning",
					Params: []v1beta1.Param{{
						Name: "greeting", Value: v1beta1.ArrayOrString{StringVal: "$(params.greeting)", Type: v1beta1.ParamTypeString}}},
					TaskSpec: &v1beta1.EmbeddedTask{
						TaskSpec: v1beta1.TaskSpec{
							Steps: []v1beta1.Step{{
								Container: v1.Container{
									Name:  "echo",
									Image: "ubuntu",
								},
							}},
						},
					},
				}},
			},
			Timeouts: &v1beta1.TimeoutFields{Pipeline: &metav1.Duration{Duration: time.Duration(15 * time.Second)}},
		},
	}
	if d := cmp.Diff(expected, cpr); d != "" {
		t.Errorf("didn't get expected CPR: %s", d)
	}
}

func TestUpdateRunFromColocatedPipelineRun(t *testing.T) {
	now := time.Now()
	cprStatus := cprv1alpha1.ColocatedPipelineRunStatus{
		Status: duckv1.Status{
			Conditions: duckv1.Conditions{
				apis.Condition{Type: apis.ConditionSucceeded, Status: v1.ConditionFalse, Reason: "failed"},
			},
		},
		ColocatedPipelineRunStatusFields: cprv1alpha1.ColocatedPipelineRunStatusFields{
			CompletionTime: &metav1.Time{Time: now},
		}}

	statusBytes, err := json.Marshal(cprStatus)
	if err != nil {
		t.Fatalf("%s", err)
	}
	run := v1alpha1.Run{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Run",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-run",
			Namespace: "default",
		},
		Spec: v1alpha1.RunSpec{
			Spec: &v1alpha1.EmbeddedRunSpec{
				TypeMeta: runtime.TypeMeta{
					Kind:       "ColocatedPipelineRun",
					APIVersion: "tekton.dev/v1alpha1",
				},
				Metadata: v1beta1.PipelineTaskMetadata{
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: runtime.RawExtension{
					Raw: []byte{},
				},
			},
		},
		Status: v1alpha1.RunStatus{
			RunStatusFields: v1alpha1.RunStatusFields{
				ExtraFields: runtime.RawExtension{
					Raw: []byte{},
				},
			},
		},
	}

	cpr := cprv1alpha1.ColocatedPipelineRun{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ColocatedPipelineRun",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				"foo2": "bar2",
			},
			Namespace: "default",
			Name:      "my-run",
		},
		Status: cprStatus,
	}

	err = pipelineinpod.UpdateRunFromColocatedPipelineRun(&run, cpr)
	expected := v1alpha1.Run{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Run",
			APIVersion: "tekton.dev/v1alpha1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-run",
			Namespace: "default",
		},
		Spec: v1alpha1.RunSpec{
			Spec: &v1alpha1.EmbeddedRunSpec{
				TypeMeta: runtime.TypeMeta{
					Kind:       "ColocatedPipelineRun",
					APIVersion: "tekton.dev/v1alpha1",
				},
				Metadata: v1beta1.PipelineTaskMetadata{
					Labels: map[string]string{
						"foo": "bar",
					},
				},
				Spec: runtime.RawExtension{
					Raw: []byte{},
				},
			},
		},
		Status: v1alpha1.RunStatus{
			Status: cprStatus.Status,
			RunStatusFields: v1alpha1.RunStatusFields{
				ExtraFields: runtime.RawExtension{
					Raw: statusBytes,
				},
				CompletionTime: &metav1.Time{Time: now},
			},
		},
	}
	if err != nil {
		t.Errorf("got err %s", err)
	}
	if diff := cmp.Diff(expected, run); diff != "" {
		t.Errorf("got != want: %s", diff)
	}
}
