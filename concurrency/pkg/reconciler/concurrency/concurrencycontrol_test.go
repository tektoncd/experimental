package concurrency_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/concurrency/pkg/apis/concurrency/v1alpha1"
	"github.com/tektoncd/experimental/concurrency/pkg/apis/config"
	fakeconcurrencyclient "github.com/tektoncd/experimental/concurrency/pkg/client/injection/client/fake"
	fakeconcurrencycontrolinformer "github.com/tektoncd/experimental/concurrency/pkg/client/injection/informers/concurrency/v1alpha1/concurrencycontrol/fake"
	"github.com/tektoncd/experimental/concurrency/pkg/reconciler/concurrency"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	ttesting "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	"github.com/tektoncd/pipeline/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ktesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/record"
	"knative.dev/pkg/apis"
	duckv1beta1 "knative.dev/pkg/apis/duck/v1beta1"
	cminformer "knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"

	_ "knative.dev/pkg/system/testing" // Setup system.Namespace()
)

// initiailizeControllerAssets is a shared helper for controller initialization.
func initializeControllerAssets(t *testing.T, d test.Data, ccs []*v1alpha1.ConcurrencyControl) (test.Assets, func()) {
	t.Helper()
	ctx, _ := ttesting.SetupFakeContext(t)
	ctx, cancel := context.WithCancel(ctx)
	ensureConfigMapsExist(&d)

	// Set up all Tekton Pipelines test data objects
	c, informers := test.SeedTestData(t, ctx, d)

	// Set up all concurrency controls
	ccInformer := fakeconcurrencycontrolinformer.Get(ctx).Informer()
	ccClient := fakeconcurrencyclient.Get(ctx)
	ccClient.PrependReactor("*", "concurrencycontrols", test.AddToInformer(t, ccInformer.GetIndexer()))
	for _, cc := range ccs {
		cc := cc.DeepCopy() // Avoid assumptions that the informer's copy is modified.
		if _, err := ccClient.CustomV1alpha1().ConcurrencyControls(cc.Namespace).Create(ctx, cc, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}

	configMapWatcher := cminformer.NewInformedWatcher(c.Kube, config.ConcurrencyNamespace)
	ctl := concurrency.NewController()(ctx, configMapWatcher)
	if la, ok := ctl.Reconciler.(reconciler.LeaderAware); ok {
		if err := la.Promote(reconciler.UniversalBucket(), func(reconciler.Bucket, types.NamespacedName) {}); err != nil {
			t.Fatalf("error promoting reconciler leader: %v", err)
		}
	}
	if err := configMapWatcher.Start(ctx.Done()); err != nil {
		t.Fatalf("error starting configmap watcher: %v", err)
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

func ensureConfigMapsExist(d *test.Data) {
	var configExists bool
	for _, cm := range d.ConfigMaps {
		if cm.Name == config.ConcurrencyConfigMapName {
			configExists = true
		}
	}
	if !configExists {
		d.ConfigMaps = append(d.ConfigMaps, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: config.ConcurrencyConfigMapName, Namespace: config.ConcurrencyNamespace},
			Data:       map[string]string{},
		})
	}
}

type concurrencyTest struct {
	test.Data  `json:"inline"`
	Test       *testing.T
	TestAssets test.Assets
	Cancel     func()
}

func newTest(data test.Data, ccs []*v1alpha1.ConcurrencyControl, t *testing.T) *concurrencyTest {
	t.Helper()
	testAssets, cancel := initializeControllerAssets(t, data, ccs)
	return &concurrencyTest{
		Data:       data,
		Test:       t,
		TestAssets: testAssets,
		Cancel:     cancel,
	}
}

func newPipelineRunSpecWithStatus(status v1beta1.PipelineRunSpecStatus) v1beta1.PipelineRunSpec {
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
	spec.Status = status
	return spec
}

func TestMatches(t *testing.T) {
	tcs := []struct {
		name      string
		pr        *v1beta1.PipelineRun
		cc        *v1alpha1.ConcurrencyControl
		wantMatch bool
	}{{
		name:      "empty selector matches everything",
		pr:        &v1beta1.PipelineRun{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"foo": "bar"}}},
		cc:        &v1alpha1.ConcurrencyControl{Spec: v1alpha1.ConcurrencySpec{}},
		wantMatch: true,
	}, {
		name:      "label matches selector",
		pr:        &v1beta1.PipelineRun{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"foo": "bar"}}},
		cc:        &v1alpha1.ConcurrencyControl{Spec: v1alpha1.ConcurrencySpec{Selector: metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}}}},
		wantMatch: true,
	}, {
		name:      "label doesn't match selector",
		pr:        &v1beta1.PipelineRun{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"foo": "abcd"}}},
		cc:        &v1alpha1.ConcurrencyControl{Spec: v1alpha1.ConcurrencySpec{Selector: metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}}}},
		wantMatch: false,
	}, {
		name:      "one label matches selector, one doesn't",
		pr:        &v1beta1.PipelineRun{ObjectMeta: metav1.ObjectMeta{Labels: map[string]string{"foo": "bar", "abc": "123"}}},
		cc:        &v1alpha1.ConcurrencyControl{Spec: v1alpha1.ConcurrencySpec{Selector: metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}}}},
		wantMatch: true,
	}}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got := concurrency.Matches(tc.pr, tc.cc)
			if tc.wantMatch != got {
				t.Errorf("wantMatch is %t but got %t", tc.wantMatch, got)
			}
		})
	}
}

func TestConcurrency(t *testing.T) {
	tcs := []struct {
		name                string
		concurrencyControls []*v1alpha1.ConcurrencyControl
		wantErr             bool
		// Labels and status for the PipelineRun being reconciled
		labels     map[string]string
		specStatus v1beta1.PipelineRunSpecStatus
		// Other PipelineRun, if any
		otherPR *v1beta1.PipelineRun
		// Expected labels and status for the PipelineRun being reconciled
		wantLabels     map[string]string
		wantSpecStatus v1beta1.PipelineRunSpecStatus
		// Expected status of the other PipelineRun
		wantOtherPRSpecStatus v1beta1.PipelineRunSpecStatus
	}{{
		name:       "no other PRs, no matching controls, controls not applied yet",
		labels:     map[string]string{"tekton.dev/ok-to-start": "true"},
		specStatus: v1beta1.PipelineRunSpecStatusPending,
		wantLabels: map[string]string{"tekton.dev/concurrency": "true"},
	}, {
		name:           "controls already applied",
		labels:         map[string]string{"tekton.dev/concurrency": "true"},
		specStatus:     v1beta1.PipelineRunSpecStatusPending,
		wantLabels:     map[string]string{"tekton.dev/concurrency": "true"},
		wantSpecStatus: v1beta1.PipelineRunSpecStatusPending,
	}, {
		name: "non pending PipelineRun is ignored",
	}, {
		name:           "user started PipelineRun as pending",
		specStatus:     v1beta1.PipelineRunSpecStatusPending,
		wantLabels:     map[string]string{"tekton.dev/concurrency": "true"},
		wantSpecStatus: v1beta1.PipelineRunSpecStatusPending,
	}, {
		name:       "no other PRs, one matching control",
		labels:     map[string]string{"tekton.dev/ok-to-start": "true", "tekton.dev/pipeline": "pipeline-run"},
		specStatus: v1beta1.PipelineRunSpecStatusPending,
		concurrencyControls: []*v1alpha1.ConcurrencyControl{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "concurrency-control",
				Namespace: "default",
			},
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "Cancel",
				Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
			},
		}},
		wantLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true"},
	}, {
		name:       "one matching control, other PR in different namespace with same key",
		labels:     map[string]string{"tekton.dev/ok-to-start": "true", "tekton.dev/pipeline": "pipeline-run"},
		specStatus: v1beta1.PipelineRunSpecStatusPending,
		concurrencyControls: []*v1alpha1.ConcurrencyControl{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "concurrency-control",
				Namespace: "default",
			},
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "Cancel",
				Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
			},
		}},
		otherPR: &v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anything",
				Namespace: "different-ns",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true"},
			},
			Spec: newPipelineRunSpecWithStatus(""),
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionUnknown,
					}},
				},
			},
		},
		wantLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true"},
	}, {
		name:       "one matching control, running PR in same namespace with same key",
		labels:     map[string]string{"tekton.dev/ok-to-start": "true", "tekton.dev/pipeline": "pipeline-run"},
		specStatus: v1beta1.PipelineRunSpecStatusPending,
		concurrencyControls: []*v1alpha1.ConcurrencyControl{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "concurrency-control",
				Namespace: "default",
			},
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "Cancel",
				Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
			},
		}},
		otherPR: &v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anything",
				Namespace: "default",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true"},
			},
			Spec: newPipelineRunSpecWithStatus(""),
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionUnknown,
					}},
				},
			},
		},
		wantLabels:            map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true"},
		wantOtherPRSpecStatus: v1beta1.PipelineRunSpecStatusCancelled,
	}, {
		name:       "one matching control, running PR in same namespace with same key, strategy = gracefully cancel",
		labels:     map[string]string{"tekton.dev/ok-to-start": "true", "tekton.dev/pipeline": "pipeline-run"},
		specStatus: v1beta1.PipelineRunSpecStatusPending,
		concurrencyControls: []*v1alpha1.ConcurrencyControl{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "concurrency-control",
				Namespace: "default",
			},
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "GracefullyCancel",
				Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
			},
		}},
		otherPR: &v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anything",
				Namespace: "default",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true"},
			},
			Spec: newPipelineRunSpecWithStatus(""),
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionUnknown,
					}},
				},
			},
		},
		wantLabels:            map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true"},
		wantOtherPRSpecStatus: v1beta1.PipelineRunSpecStatusCancelledRunFinally,
	}, {
		name:       "one matching control, running PR in same namespace with same key, strategy = gracefully stop",
		labels:     map[string]string{"tekton.dev/ok-to-start": "true", "tekton.dev/pipeline": "pipeline-run"},
		specStatus: v1beta1.PipelineRunSpecStatusPending,
		concurrencyControls: []*v1alpha1.ConcurrencyControl{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "concurrency-control",
				Namespace: "default",
			},
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "GracefullyStop",
				Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
			},
		}},
		otherPR: &v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anything",
				Namespace: "default",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true"},
			},
			Spec: newPipelineRunSpecWithStatus(""),
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionUnknown,
					}},
				},
			},
		},
		wantLabels:            map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true"},
		wantOtherPRSpecStatus: v1beta1.PipelineRunSpecStatusStoppedRunFinally,
	}, {
		name:       "one matching control, running PR in same namespace with same key and same groupby",
		labels:     map[string]string{"tekton.dev/ok-to-start": "true", "tekton.dev/pipeline": "pipeline-run", "another-label": "foobar"},
		specStatus: v1beta1.PipelineRunSpecStatusPending,
		concurrencyControls: []*v1alpha1.ConcurrencyControl{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "concurrency-control",
				Namespace: "default",
			},
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "Cancel",
				Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
				GroupBy:  []string{"another-label"},
			},
		}},
		otherPR: &v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anything",
				Namespace: "default",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true", "another-label": "foobar"},
			},
			Spec: newPipelineRunSpecWithStatus(""),
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionUnknown,
					}},
				},
			},
		},
		wantLabels:            map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true", "another-label": "foobar"},
		wantOtherPRSpecStatus: v1beta1.PipelineRunSpecStatusCancelled,
	}, {
		name:       "one matching control, running PR in same namespace with same key and different groupby",
		labels:     map[string]string{"tekton.dev/ok-to-start": "true", "tekton.dev/pipeline": "pipeline-run", "another-label": "foobar"},
		specStatus: v1beta1.PipelineRunSpecStatusPending,
		concurrencyControls: []*v1alpha1.ConcurrencyControl{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "concurrency-control",
				Namespace: "default",
			},
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "Cancel",
				Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
				GroupBy:  []string{"another-label"},
			},
		}},
		otherPR: &v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anything",
				Namespace: "default",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true", "another-label": "abcd"},
			},
			Spec: newPipelineRunSpecWithStatus(""),
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionUnknown,
					}},
				},
			},
		},
		wantLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true", "another-label": "foobar"},
	}, {
		name:       "one matching control, running PR in same namespace with same key and both missing value for groupby",
		labels:     map[string]string{"tekton.dev/ok-to-start": "true", "tekton.dev/pipeline": "pipeline-run"},
		specStatus: v1beta1.PipelineRunSpecStatusPending,
		concurrencyControls: []*v1alpha1.ConcurrencyControl{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "concurrency-control",
				Namespace: "default",
			},
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "Cancel",
				Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
				GroupBy:  []string{"another-label"},
			},
		}},
		otherPR: &v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anything",
				Namespace: "default",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true"},
			},
			Spec: newPipelineRunSpecWithStatus(""),
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionUnknown,
					}},
				},
			},
		},
		wantLabels:            map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true"},
		wantOtherPRSpecStatus: v1beta1.PipelineRunSpecStatusCancelled,
	}, {
		name:       "one matching control, running PR in same namespace with same key and existing missing value for groupby",
		labels:     map[string]string{"tekton.dev/ok-to-start": "true", "tekton.dev/pipeline": "pipeline-run"},
		specStatus: v1beta1.PipelineRunSpecStatusPending,
		concurrencyControls: []*v1alpha1.ConcurrencyControl{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "concurrency-control",
				Namespace: "default",
			},
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "Cancel",
				Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
				GroupBy:  []string{"another-label"},
			},
		}},
		otherPR: &v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anything",
				Namespace: "default",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true", "another-label": "abcd"},
			},
			Spec: newPipelineRunSpecWithStatus(""),
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionUnknown,
					}},
				},
			},
		},
		wantLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true"},
	}, {
		name:       "one matching control, running PR in same namespace with same key and other PR missing value for groupby",
		labels:     map[string]string{"tekton.dev/ok-to-start": "true", "tekton.dev/pipeline": "pipeline-run", "another-label": "foobar"},
		specStatus: v1beta1.PipelineRunSpecStatusPending,
		concurrencyControls: []*v1alpha1.ConcurrencyControl{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "concurrency-control",
				Namespace: "default",
			},
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "Cancel",
				Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
				GroupBy:  []string{"another-label"},
			},
		}},
		otherPR: &v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anything",
				Namespace: "default",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true"},
			},
			Spec: newPipelineRunSpecWithStatus(""),
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionUnknown,
					}},
				},
			},
		},
		wantLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true", "another-label": "foobar"},
	}, {
		name:       "one matching control, running PR in same namespace with same key, multiple matching groupby",
		labels:     map[string]string{"tekton.dev/ok-to-start": "true", "tekton.dev/pipeline": "pipeline-run", "another-label": "foobar", "third-label": "abcd"},
		specStatus: v1beta1.PipelineRunSpecStatusPending,
		concurrencyControls: []*v1alpha1.ConcurrencyControl{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "concurrency-control",
				Namespace: "default",
			},
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "Cancel",
				Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
				GroupBy:  []string{"another-label", "third-label"},
			},
		}},
		otherPR: &v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anything",
				Namespace: "default",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true", "another-label": "foobar", "third-label": "abcd"},
			},
			Spec: newPipelineRunSpecWithStatus(""),
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionUnknown,
					}},
				},
			},
		},
		wantLabels:            map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true", "another-label": "foobar", "third-label": "abcd"},
		wantOtherPRSpecStatus: v1beta1.PipelineRunSpecStatusCancelled,
	}, {
		name:       "one matching control, running PR in same namespace with same key, multiple groupby",
		labels:     map[string]string{"tekton.dev/ok-to-start": "true", "tekton.dev/pipeline": "pipeline-run", "another-label": "foobar", "third-label": "abcd"},
		specStatus: v1beta1.PipelineRunSpecStatusPending,
		concurrencyControls: []*v1alpha1.ConcurrencyControl{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "concurrency-control",
				Namespace: "default",
			},
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "Cancel",
				Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
				GroupBy:  []string{"another-label", "third-label"},
			},
		}},
		otherPR: &v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anything",
				Namespace: "default",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true", "another-label": "foobar", "third-label": "1234"},
			},
			Spec: newPipelineRunSpecWithStatus(""),
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionUnknown,
					}},
				},
			},
		},
		wantLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true", "another-label": "foobar", "third-label": "abcd"},
	}, {
		name:       "one matching control, pending PR in same namespace with same key",
		labels:     map[string]string{"tekton.dev/ok-to-start": "true", "tekton.dev/pipeline": "pipeline-run"},
		specStatus: v1beta1.PipelineRunSpecStatusPending,
		concurrencyControls: []*v1alpha1.ConcurrencyControl{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "concurrency-control",
				Namespace: "default",
			},
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "Cancel",
				Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
			},
		}},
		otherPR: &v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anything",
				Namespace: "default",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true"},
			},
			Spec: newPipelineRunSpecWithStatus(v1beta1.PipelineRunSpecStatusPending),
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionUnknown,
					}},
				},
			},
		},
		wantLabels:            map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true"},
		wantOtherPRSpecStatus: v1beta1.PipelineRunSpecStatusCancelled,
	}, {
		name:       "one matching control, completed PR in same namespace with same key",
		labels:     map[string]string{"tekton.dev/ok-to-start": "true", "tekton.dev/pipeline": "pipeline-run"},
		specStatus: v1beta1.PipelineRunSpecStatusPending,
		concurrencyControls: []*v1alpha1.ConcurrencyControl{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "concurrency-control",
				Namespace: "default",
			},
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "Cancel",
				Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
			},
		}},
		otherPR: &v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anything",
				Namespace: "default",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true"},
			},
			Spec: newPipelineRunSpecWithStatus(""),
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionTrue,
					}},
				},
			},
		},
		wantLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true"},
	}, {
		name:       "two matching controls, running PR in same namespace with one of same key",
		labels:     map[string]string{"tekton.dev/ok-to-start": "true", "tekton.dev/pipeline": "pipeline-run", "anotherlabel": "anotherlabelvalue"},
		specStatus: v1beta1.PipelineRunSpecStatusPending,
		concurrencyControls: []*v1alpha1.ConcurrencyControl{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "concurrency-control",
				Namespace: "default",
			},
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "Cancel",
				Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
			},
		}, {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "concurrency-control2",
				Namespace: "default",
			},
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "Cancel",
				Selector: metav1.LabelSelector{MatchLabels: map[string]string{"anotherlabel": "anotherlabelvalue"}},
			},
		}},
		otherPR: &v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anything",
				Namespace: "default",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true"},
			},
			Spec: newPipelineRunSpecWithStatus(""),
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionUnknown,
					}},
				},
			},
		},
		wantLabels:            map[string]string{"tekton.dev/pipeline": "pipeline-run", "anotherlabel": "anotherlabelvalue", "tekton.dev/concurrency": "true"},
		wantOtherPRSpecStatus: v1beta1.PipelineRunSpecStatusCancelled,
	}, {
		name:       "two matching controls with different strategies, running PR in same namespace with one of same key",
		labels:     map[string]string{"tekton.dev/ok-to-start": "true", "tekton.dev/pipeline": "pipeline-run", "anotherlabel": "anotherlabelvalue"},
		specStatus: v1beta1.PipelineRunSpecStatusPending,
		concurrencyControls: []*v1alpha1.ConcurrencyControl{{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "concurrency-control",
				Namespace: "default",
			},
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "GracefullyCancel",
				Selector: metav1.LabelSelector{MatchLabels: map[string]string{"tekton.dev/pipeline": "pipeline-run"}},
			},
		}, {
			ObjectMeta: metav1.ObjectMeta{
				Name:      "concurrency-control2",
				Namespace: "default",
			},
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "Cancel",
				Selector: metav1.LabelSelector{MatchLabels: map[string]string{"anotherlabel": "anotherlabelvalue"}},
			},
		}},
		otherPR: &v1beta1.PipelineRun{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "anything",
				Namespace: "default",
				Labels:    map[string]string{"tekton.dev/pipeline": "pipeline-run", "tekton.dev/concurrency": "true"},
			},
			Spec: newPipelineRunSpecWithStatus(""),
			Status: v1beta1.PipelineRunStatus{
				Status: duckv1beta1.Status{
					Conditions: []apis.Condition{{
						Type:   apis.ConditionSucceeded,
						Status: corev1.ConditionUnknown,
					}},
				},
			},
		},
		wantErr: true,
	}}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			namespace := "default"
			name := "pipeline-run"
			prToTest := v1beta1.PipelineRun{
				ObjectMeta: metav1.ObjectMeta{
					Labels:    tc.labels,
					Name:      name,
					Namespace: namespace,
				},
				Spec: newPipelineRunSpecWithStatus(tc.specStatus),
			}
			prs := []*v1beta1.PipelineRun{&prToTest}
			if tc.otherPR != nil {
				prs = append(prs, tc.otherPR)
			}
			prt := newTest(test.Data{PipelineRuns: prs}, tc.concurrencyControls, t)
			defer prt.Cancel()

			c := prt.TestAssets.Controller
			clients := prt.TestAssets.Clients
			reconcileError := c.Reconciler.Reconcile(prt.TestAssets.Ctx, fmt.Sprintf("%s/%s", namespace, name))
			if (reconcileError != nil) != tc.wantErr {
				t.Errorf("wantErr was %t but got reconcile err %s", tc.wantErr, reconcileError)
			}
			if tc.wantErr == false {
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

func TestConcurrencyMultipleOtherPRs(t *testing.T) {
	name := "pipeline-run"
	namespace := "default"
	prToTest := v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    map[string]string{"tekton.dev/ok-to-start": "true", "foo": "bar", "abc": "123"},
			Name:      name,
			Namespace: namespace,
		},
		Spec: newPipelineRunSpecWithStatus(v1beta1.PipelineRunSpecStatusPending),
	}
	otherPR1 := v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    map[string]string{"tekton.dev/concurrency": "true", "foo": "bar"},
			Name:      "pipeline-run1",
			Namespace: namespace,
		},
		Spec: newPipelineRunSpecWithStatus(""),
	}
	otherPR2 := v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    map[string]string{"tekton.dev/concurrency": "true", "abc": "123"},
			Name:      "pipeline-run2",
			Namespace: namespace,
		},
		Spec: newPipelineRunSpecWithStatus(""),
	}
	prs := []*v1beta1.PipelineRun{&prToTest, &otherPR1, &otherPR2}
	ccs := []*v1alpha1.ConcurrencyControl{{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "concurrency-control",
			Namespace: "default",
		},
		Spec: v1alpha1.ConcurrencySpec{
			Strategy: "Cancel",
			Selector: metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
		},
	}, {
		ObjectMeta: metav1.ObjectMeta{
			Name:      "concurrency-control2",
			Namespace: "default",
		},
		Spec: v1alpha1.ConcurrencySpec{
			Strategy: "Cancel",
			Selector: metav1.LabelSelector{MatchLabels: map[string]string{"abc": "123"}},
		},
	}}

	prt := newTest(test.Data{PipelineRuns: prs}, ccs, t)
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
	if d := cmp.Diff(v1beta1.PipelineRunSpecStatus(""), reconciledRun.Spec.Status); d != "" {
		t.Errorf("wrong spec status: %s", d)
	}
	wantLabels := map[string]string{"tekton.dev/concurrency": "true", "foo": "bar", "abc": "123"}
	if d := cmp.Diff(wantLabels, reconciledRun.Labels); d != "" {
		t.Errorf("wrong labels: %s", d)
	}

	gotOtherPR1, err := clients.Pipeline.TektonV1beta1().PipelineRuns(otherPR1.Namespace).Get(prt.TestAssets.Ctx, otherPR1.Name, metav1.GetOptions{})
	if err != nil {
		prt.Test.Fatalf("Somehow had error getting reconciled run %s out of fake client: %s", otherPR1.Name, err)
	}
	if gotOtherPR1.Spec.Status != v1beta1.PipelineRunSpecStatusCancelled {
		t.Errorf("expected PipelineRun %s to be canceled but was %s", otherPR1.Name, gotOtherPR1.Spec.Status)
	}
	gotOtherPR2, err := clients.Pipeline.TektonV1beta1().PipelineRuns(otherPR2.Namespace).Get(prt.TestAssets.Ctx, otherPR2.Name, metav1.GetOptions{})
	if err != nil {
		prt.Test.Fatalf("Somehow had error getting reconciled run %s out of fake client: %s", otherPR2.Name, err)
	}
	if gotOtherPR2.Spec.Status != v1beta1.PipelineRunSpecStatusCancelled {
		t.Errorf("expected PipelineRun %s to be canceled but was %s", otherPR2.Name, gotOtherPR2.Spec.Status)
	}
}

func TestConcurrencyWithError(t *testing.T) {
	namespace := "default"
	name := "pipeline-run"
	prToTest := v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    map[string]string{"tekton.dev/ok-to-start": "true", "foo": "bar", "abc": "123"},
			Name:      name,
			Namespace: namespace,
		},
		Spec: newPipelineRunSpecWithStatus(v1beta1.PipelineRunSpecStatusPending),
	}
	otherPR := v1beta1.PipelineRun{
		ObjectMeta: metav1.ObjectMeta{
			Labels:    map[string]string{"tekton.dev/concurrency": "true", "foo": "bar"},
			Name:      "other-pipeline-run",
			Namespace: namespace,
		},
		Spec: newPipelineRunSpecWithStatus(""),
	}
	cc := v1alpha1.ConcurrencyControl{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "concurrency-control",
			Namespace: "default",
		},
		Spec: v1alpha1.ConcurrencySpec{
			Strategy: "Cancel",
			Selector: metav1.LabelSelector{MatchLabels: map[string]string{"foo": "bar"}},
		},
	}

	prt := newTest(test.Data{PipelineRuns: []*v1beta1.PipelineRun{&prToTest, &otherPR}}, []*v1alpha1.ConcurrencyControl{&cc}, t)
	defer prt.Cancel()

	c := prt.TestAssets.Controller
	clients := prt.TestAssets.Clients
	clients.Pipeline.PrependReactor("patch", "pipelineruns", func(action ktesting.Action) (bool, runtime.Object, error) {
		return true, nil, errors.New("couldn't patch pipelinerun")
	})
	reconcileError := c.Reconciler.Reconcile(prt.TestAssets.Ctx, fmt.Sprintf("%s/%s", namespace, name))
	if reconcileError == nil {
		t.Errorf("expected reconciler error but none received")
	}
}
