package v1alpha1

import (
	"github.com/google/go-cmp/cmp"
	pipelinev1beta1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/ptr"
	"testing"
)

func TestWorkflow_ToPipelineRun(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   Workflow
		want *pipelinev1beta1.PipelineRun
	}{{
		name: "convert basic workflow spec to PR",
		in: Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-workflow",
				Namespace: "my-namespace",
			},
			Spec: WorkflowSpec{
				Params: []pipelinev1beta1.Param{{
					Name: "clone_sha",
					Value: pipelinev1beta1.ArrayOrString{
						Type:      pipelinev1beta1.ParamTypeString,
						StringVal: "2aafa87e7cd14aef64956eba19721ce2fe814536",
					},
				}},
				Pipeline: PipelineRef{
					Spec: pipelinev1beta1.PipelineSpec{
						Tasks: []pipelinev1beta1.PipelineTask{{
							Name: "clone-repo",
							TaskRef: &pipelinev1beta1.TaskRef{
								Name: "git-clone",
								Kind: "Task",
							},
						}},
						Params: []pipelinev1beta1.ParamSpec{{
							Name:        "clone_sha",
							Type:        pipelinev1beta1.ParamTypeString,
							Description: "Commit SHA to clone",
							Default:     nil,
						}},
						Workspaces: nil,
					},
				},
				ServiceAccountName: ptr.String("my-sa"),
			},
		},
		want: &pipelinev1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				GenerateName: "my-workflow-run-",
				Namespace:    "my-namespace",
			},
			Spec: pipelinev1beta1.PipelineRunSpec{
				PipelineSpec: &pipelinev1beta1.PipelineSpec{
					Tasks: []pipelinev1beta1.PipelineTask{{
						Name: "clone-repo",
						TaskRef: &pipelinev1beta1.TaskRef{
							Name: "git-clone",
							Kind: "Task",
						},
					}},
					Params: []pipelinev1beta1.ParamSpec{{
						Name:        "clone_sha",
						Type:        pipelinev1beta1.ParamTypeString,
						Description: "Commit SHA to clone",
						Default:     nil,
					}},
					Workspaces: nil,
				},
				Params: []pipelinev1beta1.Param{{
					Name: "clone_sha",
					Value: pipelinev1beta1.ArrayOrString{
						Type:      pipelinev1beta1.ParamTypeString,
						StringVal: "2aafa87e7cd14aef64956eba19721ce2fe814536",
					},
				}},
				ServiceAccountName: "my-sa",
				Timeouts:           &pipelinev1beta1.TimeoutFields{},
			},
		},
	}} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.in.ToPipelineRun()
			if err != nil {
				t.Fatalf("ToPipelineRun() err: %s", err)
			}
			if diff := cmp.Diff(tc.want, got); diff != "" {
				t.Fatalf("ToPipelineRun() -want/+got: %s", diff)
			}
		})
	}
}
