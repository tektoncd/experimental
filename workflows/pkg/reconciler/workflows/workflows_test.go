package workflows_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
	fakeworkflowsclientset "github.com/tektoncd/experimental/workflows/pkg/client/clientset/versioned/fake"
	fakeworkflowsclient "github.com/tektoncd/experimental/workflows/pkg/client/injection/client/fake"
	fakerepoinformer "github.com/tektoncd/experimental/workflows/pkg/client/injection/informers/workflows/v1alpha1/gitrepository/fake"
	fakeworkflowsinformer "github.com/tektoncd/experimental/workflows/pkg/client/injection/informers/workflows/v1alpha1/workflow/fake"
	"github.com/tektoncd/experimental/workflows/pkg/reconciler/workflows"
	"github.com/tektoncd/experimental/workflows/test/parse"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/test"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"knative.dev/pkg/apis"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	cminformer "knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/system"
)

// initiailizeControllerAssets is a shared helper for controller initialization.
func initializeControllerAssets(t *testing.T, r test.Resources, ws []*v1alpha1.Workflow, grs []*v1alpha1.GitRepository) (testAssets, func()) {
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
		if _, err := workflowsClient.WorkflowsV1alpha1().Workflows(w.Namespace).Create(context.Background(), w, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}

	reposInformer := fakerepoinformer.Get(ctx)
	for _, gr := range grs {
		gr := gr.DeepCopy()
		if err := reposInformer.Informer().GetIndexer().Add(gr); err != nil {
			t.Fatal(err)
		}
		if _, err := workflowsClient.WorkflowsV1alpha1().GitRepositories(gr.Namespace).Create(context.Background(), gr, metav1.CreateOptions{}); err != nil {
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
	return testAssets{
		Assets: test.Assets{
			Clients:    clients,
			Controller: ctl,
		},
		WorkflowsClient: workflowsClient,
	}, cancel
}

type workflowsTest struct {
	test.Resources `json:"inline"`
	Test           *testing.T
	TestAssets     testAssets
	Cancel         func()
}

type testAssets struct {
	test.Assets
	WorkflowsClient *fakeworkflowsclientset.Clientset
}

func newTest(r test.Resources, wfs []*v1alpha1.Workflow, repos []*v1alpha1.GitRepository, t *testing.T) *workflowsTest {
	t.Helper()
	testAssets, cancel := initializeControllerAssets(t, r, wfs, repos)
	return &workflowsTest{
		Resources:  r,
		Test:       t,
		TestAssets: testAssets,
		Cancel:     cancel,
	}
}

var (
	ignoreTypeMeta           = cmpopts.IgnoreFields(metav1.TypeMeta{}, "Kind", "APIVersion")
	ignoreSpec               = cmpopts.IgnoreFields(v1beta1.Trigger{}, "Spec")
	ignoreLastTransitionTime = cmpopts.IgnoreFields(apis.Condition{}, "LastTransitionTime")
)

func sortTriggers(i, j v1beta1.Trigger) bool {
	return i.Name < j.Name
}

func sortRepos(i, j v1alpha1.GitRepository) bool {
	return i.Name < j.Name
}

func TestReconcile(t *testing.T) {
	tr := true
	ownerRef := metav1.OwnerReference{APIVersion: "workflows.tekton.dev/v1alpha1", Kind: "Workflow", Name: "my-workflow", Controller: &tr, BlockOwnerDeletion: &tr}
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
				OwnerReferences: []metav1.OwnerReference{{APIVersion: "workflows.tekton.dev/v1alpha1", Kind: "Workflow", Name: "another-workflow", Controller: &tr, BlockOwnerDeletion: &tr}},
			},
		}},
		wantTriggers: []v1beta1.Trigger{{
			ObjectMeta: metav1.ObjectMeta{
				Name: "another-workflow-my-trigger-2", Namespace: namespace,
				Labels:          map[string]string{v1alpha1.WorkflowLabelKey: "another-workflow", "managed-by": "tekton-workflows"},
				OwnerReferences: []metav1.OwnerReference{{APIVersion: "workflows.tekton.dev/v1alpha1", Kind: "Workflow", Name: "another-workflow", Controller: &tr, BlockOwnerDeletion: &tr}},
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
			prt := newTest(test.Resources{Triggers: tc.existingTriggers}, []*v1alpha1.Workflow{tc.wf}, []*v1alpha1.GitRepository{}, t)
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

func TestReconcileWithRepo(t *testing.T) {
	wf := parse.MustParseWorkflow(t, "workflow", "some-namespace", `
spec:
  repos:
  - name: pipelines
  triggers:
  - name: on-pr
    event:
      types: ["pull_request"]
    bindings:
    - name: commit-sha
      value: $(body.pull_request.head.sha)
    - name: url
      value: $(body.repository.clone_url)
  pipelineSpec:
    tasks:
      - name: mytask
        taskRef:
          name: some-task
`)
	trigger := parse.MustParseTrigger(t, `
metadata:
  name: workflow-on-pr
  namespace: some-namespace
  labels:
    managed-by: tekton-workflows
    workflows.tekton.dev/workflow: workflow
  ownerReferences:
  - apiVersion: workflows.tekton.dev/v1alpha1
    kind: Workflow
    name: workflow
    controller: true
    blockOwnerDeletion: true
spec:
  name: on-pr
  interceptors:
  - name: "validate-webhook"
    ref:
      name: github
      kind: ClusterInterceptor
    params:
    - name: eventTypes
      value: ["pull_request"]
  - name: repo
    ref:
      name: cel
      kind: ClusterInterceptor
    params:
    - name: "filter"
      value:  "body.html_url.matches(https://tektoncd/pipeline)" 
  template:
    spec:
      resourcetemplates:
      - apiVersion: tekton.dev/v1beta1
        kind: PipelineRun
        metadata:
          generateName: workflow-run-
          namespace: some-namespace
        spec:
          serviceAccountName: default
          pipelineSpec:
            tasks:
            - name: mytask
              taskRef: 
                name: some-task
`)
	tcs := []struct {
		name               string
		wf                 *v1alpha1.Workflow
		wantWorkflowStatus v1alpha1.WorkflowStatus
		wantErr            bool
		existingRepos      []*v1alpha1.GitRepository
		wantRepos          []v1alpha1.GitRepository
		wantTriggers       []v1beta1.Trigger
	}{{
		name: "no existing repos; workflow fails",
		wf:   wf,
		wantWorkflowStatus: v1alpha1.WorkflowStatus{Status: duckv1.Status{Conditions: duckv1.Conditions{apis.Condition{
			Type: apis.ConditionSucceeded, Status: v1.ConditionFalse, Reason: workflows.ReasonCouldntGetRepo,
			Message: "Couldn't get repo some-namespace/pipelines: gitrepositories.workflows.tekton.dev \"pipelines\" not found"},
		}}},
		wantErr: true,
	}, {
		name: "existing repo with correct events",
		wf:   wf,
		existingRepos: []*v1alpha1.GitRepository{parse.MustParseRepo(t, "pipelines", wf.Namespace, `
spec:
  events:
  - pull_request
  url: https://github.com/tektoncd/pipeline
`)},
		wantRepos: []v1alpha1.GitRepository{*parse.MustParseRepo(t, "pipelines", wf.Namespace, `
spec:
  events:
  - pull_request
  url: https://github.com/tektoncd/pipeline
`)},
		wantTriggers: []v1beta1.Trigger{*trigger},
	}, {
		name: "existing repo with no events",
		wf:   wf,
		existingRepos: []*v1alpha1.GitRepository{parse.MustParseRepo(t, "pipelines", wf.Namespace, `
spec:
  events: []
  url: https://github.com/tektoncd/pipeline
`)},
		wantRepos: []v1alpha1.GitRepository{*parse.MustParseRepo(t, "pipelines", wf.Namespace, `
spec:
  events:
  - pull_request
  url: https://github.com/tektoncd/pipeline
`)},
		wantTriggers: []v1beta1.Trigger{*trigger},
	}, {
		name: "existing repo with extra events",
		wf:   wf,
		existingRepos: []*v1alpha1.GitRepository{parse.MustParseRepo(t, "pipelines", wf.Namespace, `
spec:
  events:
  - push
  url: https://github.com/tektoncd/pipeline
`)},
		wantRepos: []v1alpha1.GitRepository{*parse.MustParseRepo(t, "pipelines", wf.Namespace, `
spec:
  events:
  - pull_request
  - push
  url: https://github.com/tektoncd/pipeline
`)},
		wantTriggers: []v1beta1.Trigger{*trigger},
	}}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			prt := newTest(test.Resources{}, []*v1alpha1.Workflow{tc.wf}, tc.existingRepos, t)
			defer prt.Cancel()

			c := prt.TestAssets.Controller
			reconcileError := c.Reconciler.Reconcile(context.Background(), fmt.Sprintf("%s/%s", tc.wf.Namespace, tc.wf.Name))
			if (reconcileError != nil) != tc.wantErr {
				t.Errorf("unexpected reconcile err %s", reconcileError)
			}

			gotWorkflow, err := prt.TestAssets.WorkflowsClient.WorkflowsV1alpha1().Workflows(tc.wf.Namespace).Get(context.Background(), tc.wf.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("error getting workflow: %s", err)
			}
			if d := cmp.Diff(tc.wantWorkflowStatus, gotWorkflow.Status, ignoreLastTransitionTime); d != "" {
				t.Errorf("wrong workflow status: %s", d)
			}

			var gotRepos *v1alpha1.GitRepositoryList
			gotRepos, err = prt.TestAssets.WorkflowsClient.WorkflowsV1alpha1().GitRepositories(tc.wf.Namespace).List(context.Background(), metav1.ListOptions{})
			if err != nil {
				t.Fatalf("error listing repos: %s", err)
			}
			opts := []cmp.Option{cmpopts.SortSlices(sortRepos), ignoreTypeMeta, cmpopts.EquateEmpty()}
			if d := cmp.Diff(tc.wantRepos, gotRepos.Items, opts...); d != "" {
				t.Errorf("wrong repos: %s", d)
			}

			var gotTriggers *v1beta1.TriggerList
			gotTriggers, err = prt.TestAssets.Clients.Triggers.TriggersV1beta1().Triggers(tc.wf.Namespace).List(context.Background(), metav1.ListOptions{})
			if err != nil {
				t.Fatalf("error listing triggers: %s", err)
			}
			triggerOpts := []cmp.Option{cmpopts.SortSlices(sortTriggers), ignoreTypeMeta, cmpopts.EquateEmpty(), ignoreSpec}
			if d := cmp.Diff(tc.wantTriggers, gotTriggers.Items, triggerOpts...); d != "" {
				t.Errorf("wrong triggers: %s", d)
			}
		})
	}
}
