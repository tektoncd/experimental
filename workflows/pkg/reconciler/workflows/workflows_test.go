package workflows_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
	fakeworkflowsclient "github.com/tektoncd/experimental/workflows/pkg/client/injection/client/fake"
	"github.com/tektoncd/experimental/workflows/pkg/reconciler/workflows"

	fakeworkflowsinformer "github.com/tektoncd/experimental/workflows/pkg/client/injection/informers/workflows/v1alpha1/workflow/fake"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	cminformer "knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/system"
)

// initiailizeControllerAssets is a shared helper for controller initialization.
func initializeControllerAssets(t *testing.T, r test.Resources, ws []*v1alpha1.Workflow) (test.Assets, func()) {
	t.Helper()
	ctx, _ := test.SetupFakeContext(t)
	ctx, cancel := context.WithCancel(ctx)
	// Set up all Tekton Pipelines and Triggers test data objects
	clients := test.SeedResources(t, ctx, r)

	// Set up all workflows
	workflowsInformer := fakeworkflowsinformer.Get(ctx)
	workflowsClient := fakeworkflowsclient.Get(ctx)
	for _, w := range ws {
		w := w.DeepCopy()
		if err := workflowsInformer.Informer().GetIndexer().Add(w); err != nil {
			t.Fatal(err)
		}
		if _, err := workflowsClient.TektonV1alpha1().Workflows(w.Namespace).Create(context.Background(), w, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}

	configMapWatcher := cminformer.NewInformedWatcher(clients.Kube, system.Namespace())
	ctl := workflows.NewController(ctx, configMapWatcher)
	if la, ok := ctl.Reconciler.(reconciler.LeaderAware); ok {
		if err := la.Promote(reconciler.UniversalBucket(), func(reconciler.Bucket, types.NamespacedName) {}); err != nil {
			t.Fatalf("error promoting reconciler leader: %v", err)
		}
	}
	return test.Assets{
		Clients:    clients,
		Controller: ctl,
	}, cancel
}

type workflowsTest struct {
	test.Resources `json:"inline"`
	Test           *testing.T
	TestAssets     test.Assets
	Cancel         func()
}

func newTest(r test.Resources, wfs []*v1alpha1.Workflow, t *testing.T) *workflowsTest {
	t.Helper()
	testAssets, cancel := initializeControllerAssets(t, r, wfs)
	return &workflowsTest{
		Resources:  r,
		Test:       t,
		TestAssets: testAssets,
		Cancel:     cancel,
	}
}

var ignoreTypeMeta = cmpopts.IgnoreFields(metav1.TypeMeta{}, "Kind", "APIVersion")
var ignoreSpec = cmpopts.IgnoreFields(v1beta1.Trigger{}, "Spec")

func sortTriggers(i, j v1beta1.Trigger) bool {
	return i.Name < j.Name
}

func TestReconcile(t *testing.T) {
	tr := true
	ownerRef := metav1.OwnerReference{APIVersion: "tekton.dev/v1alpha1", Kind: "Workflow", Name: "my-workflow", Controller: &tr, BlockOwnerDeletion: &tr}
	namespace := "default"
	tcs := []struct {
		name             string
		wf               *v1alpha1.Workflow
		existingTriggers []*v1beta1.Trigger
		wantTriggers     []v1beta1.Trigger
	}{{
		name: "no existing triggers",
		wf: &v1alpha1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-workflow",
				Namespace: namespace,
			},
			Spec: v1alpha1.WorkflowSpec{
				Triggers: []v1alpha1.Trigger{{
					Name: "my-trigger",
				}},
			},
		},
		wantTriggers: []v1beta1.Trigger{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-workflow-my-trigger", Namespace: namespace,
				Labels:          map[string]string{v1alpha1.WorkflowLabelKey: "my-workflow", "managed-by": "tekton-workflows"},
				OwnerReferences: []metav1.OwnerReference{ownerRef},
			},
		}},
	}, {
		name: "one existing trigger, no new triggers",
		wf: &v1alpha1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-workflow",
				Namespace: namespace,
			},
			Spec: v1alpha1.WorkflowSpec{
				Triggers: []v1alpha1.Trigger{{
					Name: "my-trigger-1",
				}},
			},
		},
		existingTriggers: []*v1beta1.Trigger{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-workflow-my-trigger-1", Namespace: namespace,
				Labels:          map[string]string{v1alpha1.WorkflowLabelKey: "my-workflow", "managed-by": "tekton-workflows"},
				OwnerReferences: []metav1.OwnerReference{ownerRef},
			},
		}},
		wantTriggers: []v1beta1.Trigger{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-workflow-my-trigger-1", Namespace: namespace,
				Labels:          map[string]string{v1alpha1.WorkflowLabelKey: "my-workflow", "managed-by": "tekton-workflows"},
				OwnerReferences: []metav1.OwnerReference{ownerRef},
			},
		}},
	}, {
		name: "one existing trigger, one new trigger",
		wf: &v1alpha1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-workflow",
				Namespace: namespace,
			},
			Spec: v1alpha1.WorkflowSpec{
				Triggers: []v1alpha1.Trigger{{
					Name: "my-trigger-1",
				}, {
					Name: "my-trigger-2",
				}},
			},
		},
		existingTriggers: []*v1beta1.Trigger{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-workflow-my-trigger-1", Namespace: namespace,
				Labels:          map[string]string{v1alpha1.WorkflowLabelKey: "my-workflow", "managed-by": "tekton-workflows"},
				OwnerReferences: []metav1.OwnerReference{ownerRef},
			},
		}},
		wantTriggers: []v1beta1.Trigger{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-workflow-my-trigger-1", Namespace: namespace,
				Labels:          map[string]string{v1alpha1.WorkflowLabelKey: "my-workflow", "managed-by": "tekton-workflows"},
				OwnerReferences: []metav1.OwnerReference{ownerRef},
			},
		}, {
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-workflow-my-trigger-2", Namespace: namespace,
				Labels:          map[string]string{v1alpha1.WorkflowLabelKey: "my-workflow", "managed-by": "tekton-workflows"},
				OwnerReferences: []metav1.OwnerReference{ownerRef},
			},
		}},
	}, {
		name: "existing trigger should be deleted",
		wf: &v1alpha1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-workflow",
				Namespace: namespace,
			},
			Spec: v1alpha1.WorkflowSpec{
				Triggers: []v1alpha1.Trigger{{
					Name: "my-trigger-1",
				}},
			},
		},
		existingTriggers: []*v1beta1.Trigger{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-workflow-my-trigger-2", Namespace: namespace,
				Labels:          map[string]string{v1alpha1.WorkflowLabelKey: "my-workflow", "managed-by": "tekton-workflows"},
				OwnerReferences: []metav1.OwnerReference{ownerRef},
			},
		}},
		wantTriggers: []v1beta1.Trigger{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-workflow-my-trigger-1", Namespace: namespace,
				Labels:          map[string]string{v1alpha1.WorkflowLabelKey: "my-workflow", "managed-by": "tekton-workflows"},
				OwnerReferences: []metav1.OwnerReference{ownerRef},
			},
		}},
	}, {
		name: "existing trigger belonging to other workflow",
		wf: &v1alpha1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-workflow",
				Namespace: namespace,
			},
			Spec: v1alpha1.WorkflowSpec{
				Triggers: []v1alpha1.Trigger{{
					Name: "my-trigger-1",
				}},
			},
		},
		existingTriggers: []*v1beta1.Trigger{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "another-workflow-my-trigger-2", Namespace: namespace,
				Labels:          map[string]string{v1alpha1.WorkflowLabelKey: "another-workflow", "managed-by": "tekton-workflows"},
				OwnerReferences: []metav1.OwnerReference{{APIVersion: "tekton.dev/v1alpha1", Kind: "Workflow", Name: "another-workflow", Controller: &tr, BlockOwnerDeletion: &tr}},
			},
		}},
		wantTriggers: []v1beta1.Trigger{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "another-workflow-my-trigger-2", Namespace: namespace,
				Labels:          map[string]string{v1alpha1.WorkflowLabelKey: "another-workflow", "managed-by": "tekton-workflows"},
				OwnerReferences: []metav1.OwnerReference{{APIVersion: "tekton.dev/v1alpha1", Kind: "Workflow", Name: "another-workflow", Controller: &tr, BlockOwnerDeletion: &tr}},
			},
		}, {
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-workflow-my-trigger-1", Namespace: namespace,
				Labels:          map[string]string{v1alpha1.WorkflowLabelKey: "my-workflow", "managed-by": "tekton-workflows"},
				OwnerReferences: []metav1.OwnerReference{ownerRef},
			},
		}},
	}, {
		name: "existing trigger in different namespace",
		wf: &v1alpha1.Workflow{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "my-workflow",
				Namespace: namespace,
			},
			Spec: v1alpha1.WorkflowSpec{
				Triggers: []v1alpha1.Trigger{{
					Name: "my-trigger-1",
				}, {
					Name: "my-trigger-2",
				}},
			},
		},
		existingTriggers: []*v1beta1.Trigger{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-workflow-my-trigger-2", Namespace: "other-namespace",
				Labels:          map[string]string{v1alpha1.WorkflowLabelKey: "my-workflow", "managed-by": "tekton-workflows"},
				OwnerReferences: []metav1.OwnerReference{ownerRef},
			},
		}},
		wantTriggers: []v1beta1.Trigger{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-workflow-my-trigger-1", Namespace: namespace,
				Labels:          map[string]string{v1alpha1.WorkflowLabelKey: "my-workflow", "managed-by": "tekton-workflows"},
				OwnerReferences: []metav1.OwnerReference{ownerRef},
			},
		}, {
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-workflow-my-trigger-2", Namespace: namespace,
				Labels:          map[string]string{v1alpha1.WorkflowLabelKey: "my-workflow", "managed-by": "tekton-workflows"},
				OwnerReferences: []metav1.OwnerReference{ownerRef},
			},
		}},
	}}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			prt := newTest(test.Resources{Triggers: tc.existingTriggers}, []*v1alpha1.Workflow{tc.wf}, t)
			defer prt.Cancel()

			c := prt.TestAssets.Controller
			reconcileError := c.Reconciler.Reconcile(context.Background(), fmt.Sprintf("%s/%s", tc.wf.Namespace, tc.wf.Name))
			if reconcileError != nil {
				t.Errorf("unexpected reconcile err %s", reconcileError)
			}
			var gotTriggers *v1beta1.TriggerList
			var err error
			gotTriggers, err = prt.TestAssets.Clients.Triggers.TriggersV1beta1().Triggers(tc.wf.Namespace).List(context.Background(), metav1.ListOptions{})
			if err != nil {
				t.Fatalf("error listing triggers: %s", err)
			}
			opts := []cmp.Option{cmpopts.SortSlices(sortTriggers), ignoreTypeMeta, cmpopts.EquateEmpty(), ignoreSpec}
			if d := cmp.Diff(tc.wantTriggers, gotTriggers.Items, opts...); d != "" {
				t.Errorf("wrong triggers: %s", d)
			}
		})
	}
}
