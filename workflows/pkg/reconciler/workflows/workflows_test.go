package workflows_test

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	fluxnotifications "github.com/fluxcd/notification-controller/api/v1beta1"
	fluxsource "github.com/fluxcd/source-controller/api/v1beta2"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
	fakeworkflowsclient "github.com/tektoncd/experimental/workflows/pkg/client/injection/client/fake"
	fakeworkflowsinformer "github.com/tektoncd/experimental/workflows/pkg/client/injection/informers/workflows/v1alpha1/workflow/fake"
	"github.com/tektoncd/experimental/workflows/pkg/convert"
	"github.com/tektoncd/experimental/workflows/pkg/reconciler/workflows"
	"github.com/tektoncd/experimental/workflows/test/parse"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	"github.com/tektoncd/triggers/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	fakekubeclientset "k8s.io/client-go/kubernetes/fake"
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
	addFluxResources(clients.Kube)

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

// addFluxResources will update clientset to know it knows about the types it is
// expected to be able to interact with.
func addFluxResources(clientset *fakekubeclientset.Clientset) {
	clientset.Resources = append(clientset.Resources, &metav1.APIResourceList{
		GroupVersion: "source.toolkit.fluxcd.io/v1beta2",
		APIResources: []metav1.APIResource{{
			Group:      "source.toolkit.fluxcd.io",
			Version:    "v1beta2",
			Namespaced: true,
			Name:       "gitrepositories",
			Kind:       "GitRepository",
		}}})

	nameKind := map[string]string{
		"providers": "Provider",
		"receivers": "Receiver",
		"alerts":    "Alert",
	}
	resources := make([]metav1.APIResource, 0, len(nameKind))
	for name, kind := range nameKind {
		resources = append(resources, metav1.APIResource{
			Group:      "notification.toolkit.fluxcd.io",
			Version:    "v1beta1",
			Namespaced: true,
			Name:       name,
			Kind:       kind,
		})
	}

	clientset.Resources = append(clientset.Resources, &metav1.APIResourceList{
		GroupVersion: "notification.toolkit.fluxcd.io/v1beta1",
		APIResources: resources,
	})
}

func getFluxResource(t *testing.T, dc dynamic.Interface, gvr schema.GroupVersionResource, name string, into interface{}) {
	t.Helper()
	unsObj, err := dc.Resource(gvr).Namespace("flux-system").Get(context.Background(), name, metav1.GetOptions{})
	if err != nil {
		t.Fatalf("error getting resource %s of type %s", name, gvr.Resource)
	}
	b, err := unsObj.MarshalJSON()
	if err != nil {
		t.Fatalf("err marshaling json: %s", err)
	}
	err = json.Unmarshal(b, into)
	if err != nil {
		t.Fatalf("err unmarshaling json: %s", err)
	}
}

func verifyFluxResources(t *testing.T, dc dynamic.Interface, want convert.FluxResources) {
	t.Helper()
	for _, wantRepo := range want.Repos {
		gotRepo := fluxsource.GitRepository{}
		getFluxResource(t, dc, workflows.FluxSourceGVR, wantRepo.Name, &gotRepo)
		if d := cmp.Diff(wantRepo, gotRepo); d != "" {
			t.Errorf("wrong repo: %s", d)
		}
	}
	for _, wantReceiver := range want.Receivers {
		gotReceiver := fluxnotifications.Receiver{}
		getFluxResource(t, dc, workflows.FluxReceiverGVR, wantReceiver.Name, &gotReceiver)
		if d := cmp.Diff(wantReceiver, gotReceiver); d != "" {
			t.Errorf("wrong receiver: %s", d)
		}
	}
	if want.Provider != nil {
		gotProvider := fluxnotifications.Provider{}
		getFluxResource(t, dc, workflows.FluxProviderGVR, want.Provider.Name, &gotProvider)
		if d := cmp.Diff(want.Provider, &gotProvider); d != "" {
			t.Errorf("wrong provider: %s", d)
		}
	}
	if want.Alert != nil {
		gotAlert := fluxnotifications.Alert{}
		getFluxResource(t, dc, workflows.FluxAlertGVR, want.Alert.Name, &gotAlert)
		if d := cmp.Diff(want.Alert, &gotAlert); d != "" {
			t.Errorf("wrong alert: %s", d)
		}
	}
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

func TestReconcileWithEvents(t *testing.T) {
	provider := parse.MustParseProvider(t, "workflow", "flux-system", `
spec:
  type: generic
  address: http://el-workflows-listener.tekton-workflows.svc.cluster.local:8080/
`)
	alert := parse.MustParseAlert(t, "workflow", "flux-system", `
spec:
  providerRef:
    name: workflow
  eventSeverity: info
  eventSources:
  - kind: GitRepository
    name: workflow-pipelines
    namespace: flux-system
`)
	tcs := []struct {
		name              string
		wf                *v1alpha1.Workflow
		wantTriggers      []v1beta1.Trigger
		wantFluxResources convert.FluxResources
	}{{
		wf: parse.MustParseWorkflow(t, "workflow", "default", `
spec:
  repos:
  - name: pipelines
    url: https://tektoncd/pipeline
    vcsType: github
  triggers:
  - name: on-push-and-ping-to-main-branch
    event:
      types:
      - push
      - ping
      source:
        repo: pipelines
      secret:
        secretName: webhook-token
    filters:
      gitRef:
        regex: main
  pipeline:
    spec:
      tasks:
      - name: task-with-no-params
        taskRef:
          name: some-task
`),
		wantTriggers: []v1beta1.Trigger{
			*parse.MustParseTrigger(t, `
metadata:
  name: workflow-on-push-and-ping-to-main-branch
  namespace: default
  labels:
    managed-by: tekton-workflows
    tekton.dev/workflow: workflow
  ownerReferences:
  - apiVersion: tekton.dev/v1alpha1
    kind: Workflow
    name: workflow
    controller: true
    blockOwnerDeletion: true
spec:
  name: on-pr
  interceptors:
  - name: "filter-flux-events"
    ref:
      name: cel
      kind: ClusterInterceptor
    params:
    - name: filter
      value: "header.canonical('Gotk-Component') == 'source-controller' && body.involvedObject.kind == 'GitRepository'"
  template:
    spec:
      resourcetemplates:
      - apiVersion: tekton.dev/v1beta1
        kind: PipelineRun
        metadata:
          generateName: trigger-workflow-run-
          namespace: some-namespace
        spec:
          serviceAccountName: default
          pipelineSpec:
            tasks:
            - name: task-with-no-params
              taskRef: 
                name: some-task
`),
		},
		wantFluxResources: convert.FluxResources{
			Repos: []fluxsource.GitRepository{
				parse.MustParseRepo(t, "workflow-pipelines", "flux-system", `
spec:
  interval: 1m0s
  url: https://tektoncd/pipeline
  ref:
    branch: main
  secretRef: {}
`),
			},
			Receivers: []fluxnotifications.Receiver{
				parse.MustParseReceiver(t, "workflow-pipelines", "flux-system", `
spec:
  type: github
  events:
  - push
  - ping
  secretRef:
    name: webhook-token
  resources:
  - kind: GitRepository
    name: workflow-pipelines
    namespace: flux-system
`),
			},
			Alert:    &alert,
			Provider: &provider,
		},
	}}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			prt := newTest(test.Resources{}, []*v1alpha1.Workflow{tc.wf}, t)
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
			dc := prt.TestAssets.Clients.DynamicClient
			verifyFluxResources(t, dc, tc.wantFluxResources)
		})
	}
}
