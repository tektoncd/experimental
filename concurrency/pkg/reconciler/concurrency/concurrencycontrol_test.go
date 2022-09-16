package concurrency_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/concurrency/pkg/apis/concurrency/v1alpha1"
	fakeconcurrencyclient "github.com/tektoncd/experimental/concurrency/pkg/client/injection/client/fake"
	fakeconcurrencycontrolinformer "github.com/tektoncd/experimental/concurrency/pkg/client/injection/informers/concurrency/v1alpha1/concurrencycontrol/fake"
	"github.com/tektoncd/experimental/concurrency/pkg/reconciler/concurrency"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	ttesting "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	"github.com/tektoncd/pipeline/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	cminformer "knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/system"

	_ "knative.dev/pkg/system/testing" // Setup system.Namespace()
)

// initiailizeControllerAssets is a shared helper for controller initialization.
func initializeCControllerAssets(t *testing.T, d CCTestData, opts pipeline.Options) (test.Assets, func()) {
	t.Helper()
	ctx, _ := ttesting.SetupFakeContext(t)
	ctx, cancel := context.WithCancel(ctx)
	c, informers := test.SeedTestData(t, ctx, d.prtestData)
	ccInformer := fakeconcurrencycontrolinformer.Get(ctx).Informer()
	ccClient := fakeconcurrencyclient.Get(ctx)
	ccClient.PrependReactor("*", "concurrencycontrols", test.AddToInformer(t, ccInformer.GetIndexer()))
	for _, cc := range d.ccs {
		cc := cc.DeepCopy() // Avoid assumptions that the informer's copy is modified.
		if _, err := ccClient.ConcurrencyV1alpha1().ConcurrencyControls(cc.Namespace).Create(ctx, cc, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}
	configMapWatcher := cminformer.NewInformedWatcher(c.Kube, system.Namespace())
	ctl := concurrency.NewController(&opts)(ctx, configMapWatcher)
	if la, ok := ctl.Reconciler.(reconciler.LeaderAware); ok {
		if err := la.Promote(reconciler.UniversalBucket(), func(reconciler.Bucket, types.NamespacedName) {}); err != nil {
			t.Fatalf("error promoting reconciler leader: %v", err)
		}
	}
	return test.Assets{
		Logger:     logging.FromContext(ctx),
		Clients:    c,
		Controller: ctl,
		Informers:  informers,
		Recorder:   controller.GetEventRecorder(ctx).(*record.FakeRecorder),
		Ctx:        ctx,
	}, cancel
}

type PRT struct {
	CCTestData `json:"inline"`
	Test       *testing.T
	TestAssets test.Assets
	Cancel     func()
}

// getPipelineRunController returns an instance of the PipelineRun controller/reconciler that has been seeded with
// d, where d represents the state of the system (existing resources) needed for the test.
func getCController(t *testing.T, d CCTestData) (test.Assets, func()) {
	return initializeCControllerAssets(t, d, pipeline.Options{})
}

// newPipelineRunTest returns PipelineRunTest with a new PipelineRun controller created with specified state through data
// This PipelineRunTest can be reused for multiple PipelineRuns by calling reconcileRun for each pipelineRun
func newPRT(data CCTestData, t *testing.T) *PRT {
	t.Helper()
	testAssets, cancel := getCController(t, data)
	return &PRT{
		CCTestData: data,
		Test:       t,
		TestAssets: testAssets,
		Cancel:     cancel,
	}
}

type CCTestData struct {
	prtestData test.Data
	ccs        []*v1alpha1.ConcurrencyControl
}

func TestConcurrency2(t *testing.T) {
	spec := v1beta1.PipelineRunSpec{
		Params: []v1beta1.Param{{
			Name:  "param-1",
			Value: *v1beta1.NewArrayOrString("value-for-param"),
		}, {
			Name:  "param-2",
			Value: *v1beta1.NewArrayOrString("abcd"),
		}},
		PipelineSpec: &v1beta1.PipelineSpec{
			Params: []v1beta1.ParamSpec{{
				Name: "param-1",
			}, {
				Name: "param-2",
			}},
			Tasks: []v1beta1.PipelineTask{{
				Name: "task1",
				TaskSpec: &v1beta1.EmbeddedTask{
					TaskSpec: v1beta1.TaskSpec{
						Steps: []v1beta1.Step{{
							Image: "foo",
						}},
					},
				},
			}},
		},
	}
	tcs := []struct {
		name                  string
		concurrencyControls   []*v1alpha1.ConcurrencyControl
		labels                map[string]string
		otherPR               *v1beta1.PipelineRun
		wantSpecStatus        v1beta1.PipelineRunSpecStatus
		wantOtherPRSpecStatus v1beta1.PipelineRunSpecStatus
		wantLabels            map[string]string
	}{{
		name:       "no other PRs, no concurrency labels, no matching controls",
		wantLabels: map[string]string{"tekton.dev/concurrency": "None"},
	}, /* {
			name:       "no other PRs, existing concurrency labels",
			labels:     map[string]string{"tekton.dev/concurrency-concurrency-control": "pipeline-run"},
			wantLabels: map[string]string{"tekton.dev/concurrency-concurrency-control": "pipeline-run"},
		}, */
		{
			name:   "no other PRs, no concurrency labels, one matching control",
			labels: map[string]string{"tekton.dev/pipeline": "pipeline-run"},
			concurrencyControls: []*v1alpha1.ConcurrencyControl{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "concurrency-control",
					Namespace: "default",
				},
				Spec: v1alpha1.ConcurrencySpec{
					Params: []v1beta1.ParamSpec{{ // concurrency control params must match params declared on pipelinerun
						Name: "param-1",
					}},
					Key:      "$(params.param-1)-foo",
					Strategy: "Cancel",
					Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
				},
			}},
			wantLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency-concurrency-control": "value-for-param-foo"},
		}, {
			name:   "one matching control, other PR in different namespace with same key",
			labels: map[string]string{"tekton.dev/pipeline": "pipeline-run"},
			concurrencyControls: []*v1alpha1.ConcurrencyControl{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "concurrency-control",
					Namespace: "default",
				},
				Spec: v1alpha1.ConcurrencySpec{
					Params: []v1beta1.ParamSpec{{ // concurrency control params must match params declared on pipelinerun
						Name: "param-1",
					}},
					Key:      "$(params.param-1)-foo",
					Strategy: "Cancel",
					Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
				},
			}},
			otherPR: &v1beta1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "anything",
					Namespace: "different-ns",
					Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency-concurrency-control": "value-for-param-foo"},
				},
				Spec: spec,
				Status: v1beta1.PipelineRunStatus{
					Status: duckv1beta1.Status{
						Conditions: []apis.Condition{{
							Type:   apis.ConditionSucceeded,
							Status: corev1.ConditionUnknown,
						}},
					},
				},
			},
			wantLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency-concurrency-control": "value-for-param-foo"},
		}, {
			name:   "one matching control, running PR in same namespace with same key",
			labels: map[string]string{"tekton.dev/pipeline": "pipeline-run"},
			concurrencyControls: []*v1alpha1.ConcurrencyControl{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "concurrency-control",
					Namespace: "default",
				},
				Spec: v1alpha1.ConcurrencySpec{
					Params: []v1beta1.ParamSpec{{ // concurrency control params must match params declared on pipelinerun
						Name: "param-1",
					}},
					Key:      "$(params.param-1)-foo",
					Strategy: "Cancel",
					Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
				},
			}},
			otherPR: &v1beta1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "anything",
					Namespace: "default",
					Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency-concurrency-control": "value-for-param-foo"},
				},
				Spec: spec,
				Status: v1beta1.PipelineRunStatus{
					Status: duckv1beta1.Status{
						Conditions: []apis.Condition{{
							Type:   apis.ConditionSucceeded,
							Status: corev1.ConditionUnknown,
						}},
					},
				},
			},
			wantLabels:            map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency-concurrency-control": "value-for-param-foo"},
			wantOtherPRSpecStatus: v1beta1.PipelineRunSpecStatusCancelled,
		}, {
			name:   "one matching control, pending PR in same namespace with same key",
			labels: map[string]string{"tekton.dev/pipeline": "pipeline-run"},
			concurrencyControls: []*v1alpha1.ConcurrencyControl{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "concurrency-control",
					Namespace: "default",
				},
				Spec: v1alpha1.ConcurrencySpec{
					Params: []v1beta1.ParamSpec{{ // concurrency control params must match params declared on pipelinerun
						Name: "param-1",
					}},
					Key:      "$(params.param-1)-foo",
					Strategy: "Cancel",
					Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
				},
			}},
			otherPR: &v1beta1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "anything",
					Namespace: "default",
					Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency-concurrency-control": "value-for-param-foo"},
				},
				Spec: v1beta1.PipelineRunSpec{
					Params: []v1beta1.Param{{
						Name:  "param-1",
						Value: *v1beta1.NewArrayOrString("value-for-param"),
					}},
					Status: v1beta1.PipelineRunSpecStatusPending,
					PipelineSpec: &v1beta1.PipelineSpec{
						Params: []v1beta1.ParamSpec{{
							Name: "param-1",
						}},
						Tasks: []v1beta1.PipelineTask{{
							Name: "task1",
							TaskSpec: &v1beta1.EmbeddedTask{
								TaskSpec: v1beta1.TaskSpec{
									Steps: []v1beta1.Step{{
										Image: "foo",
									}},
								},
							},
						}},
					},
				},
				Status: v1beta1.PipelineRunStatus{
					Status: duckv1beta1.Status{
						Conditions: []apis.Condition{{
							Type:   apis.ConditionSucceeded,
							Status: corev1.ConditionUnknown,
						}},
					},
				},
			},
			wantLabels:            map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency-concurrency-control": "value-for-param-foo"},
			wantOtherPRSpecStatus: v1beta1.PipelineRunSpecStatusCancelled,
		}, {
			name:   "one matching control, completed PR in same namespace with same key",
			labels: map[string]string{"tekton.dev/pipeline": "pipeline-run"},
			concurrencyControls: []*v1alpha1.ConcurrencyControl{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "concurrency-control",
					Namespace: "default",
				},
				Spec: v1alpha1.ConcurrencySpec{
					Params: []v1beta1.ParamSpec{{ // concurrency control params must match params declared on pipelinerun
						Name: "param-1",
					}},
					Key:      "$(params.param-1)-foo",
					Strategy: "Cancel",
					Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
				},
			}},
			otherPR: &v1beta1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "anything",
					Namespace: "default",
					Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency-concurrency-control": "value-for-param-foo"},
				},
				Spec: spec,
				Status: v1beta1.PipelineRunStatus{
					Status: duckv1beta1.Status{
						Conditions: []apis.Condition{{
							Type:   apis.ConditionSucceeded,
							Status: corev1.ConditionTrue,
						}},
					},
				},
			},
			wantLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency-concurrency-control": "value-for-param-foo"},
		}, {
			name:   "two matching controls, running PR in same namespace with one of same key",
			labels: map[string]string{"tekton.dev/pipeline": "pipeline-run", "anotherlabel": "anotherlabelvalue"},
			concurrencyControls: []*v1alpha1.ConcurrencyControl{{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "concurrency-control",
					Namespace: "default",
				},
				Spec: v1alpha1.ConcurrencySpec{
					Params: []v1beta1.ParamSpec{{ // concurrency control params must match params declared on pipelinerun
						Name: "param-1",
					}},
					Key:      "$(params.param-1)-foo",
					Strategy: "Cancel",
					Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
				},
			}, {
				ObjectMeta: metav1.ObjectMeta{
					Name:      "concurrency-control2",
					Namespace: "default",
				},
				Spec: v1alpha1.ConcurrencySpec{
					Params: []v1beta1.ParamSpec{{ // concurrency control params must match params declared on pipelinerun
						Name: "param-2",
					}},
					Key:      "$(params.param-2)-foo",
					Strategy: "Cancel",
					Selector: metav1.LabelSelector{MatchLabels: map[string]string{"anotherlabel": "anotherlabelvalue"}},
				},
			}},
			otherPR: &v1beta1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "anything",
					Namespace: "default",
					Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency-concurrency-control": "value-for-param-foo"},
				},
				Spec: spec,
				Status: v1beta1.PipelineRunStatus{
					Status: duckv1beta1.Status{
						Conditions: []apis.Condition{{
							Type:   apis.ConditionSucceeded,
							Status: corev1.ConditionUnknown,
						}},
					},
				},
			},
			wantLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run", "anotherlabel": "anotherlabelvalue", "tekton.dev/concurrency-concurrency-control": "value-for-param-foo",
				"tekton.dev/concurrency-concurrency-control2": "abcd-foo"},
			wantOtherPRSpecStatus: v1beta1.PipelineRunSpecStatusCancelled,
		}}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			namespace := "default"
			name := "pipeline-run"
			spec := spec.DeepCopy()
			spec.Status = v1beta1.PipelineRunSpecStatusPending
			prToTest := v1beta1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Labels:    tc.labels,
					Name:      name,
					Namespace: namespace,
				},
				Spec: *spec,
			}
			prs := []*v1beta1.PipelineRun{&prToTest}
			if tc.otherPR != nil {
				prs = append(prs, tc.otherPR)
			}
			d := CCTestData{
				prtestData: test.Data{PipelineRuns: prs},
				ccs:        tc.concurrencyControls,
			}
			prt := newPRT(d, t)
			defer prt.Cancel()

			c := prt.TestAssets.Controller
			clients := prt.TestAssets.Clients
			reconcileError := c.Reconciler.Reconcile(prt.TestAssets.Ctx, fmt.Sprintf("%s/%s", namespace, name))
			if reconcileError != nil {
				t.Errorf("unexpected reconcile err %s", reconcileError)
			}
			// Check that the PipelineRun was reconciled correctly
			reconciledRun, err := clients.Pipeline.TektonV1beta1().PipelineRuns(namespace).Get(prt.TestAssets.Ctx, name, metav1.GetOptions{})
			if err != nil {
				prt.Test.Fatalf("Somehow had error getting reconciled run out of fake client: %s", err)
			}
			if d := cmp.Diff(tc.wantSpecStatus, reconciledRun.Spec.Status); d != "" {
				t.Errorf("wrong spec status: %s", d)
			}
			if d := cmp.Diff(tc.wantLabels, reconciledRun.Labels); d != "" {
				t.Errorf("wrong labels: %s", d)
			}

			if tc.otherPR != nil {
				otherPR, err := clients.Pipeline.TektonV1beta1().PipelineRuns(tc.otherPR.Namespace).Get(prt.TestAssets.Ctx, tc.otherPR.Name, metav1.GetOptions{})
				if err != nil {
					prt.Test.Fatalf("Somehow had error getting reconciled run %s out of fake client: %s", tc.otherPR.Name, err)
				}
				if d := cmp.Diff(tc.wantOtherPRSpecStatus, otherPR.Spec.Status); d != "" {
					t.Errorf("wrong spec status for other PR: %s", d)
				}
			}
		})
	}

}
