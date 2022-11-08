package repos_test

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
	"github.com/tektoncd/experimental/workflows/pkg/reconciler/repos"
	"github.com/tektoncd/experimental/workflows/test/parse"
	"github.com/tektoncd/triggers/test"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ghsource "knative.dev/eventing-github/pkg/apis/sources/v1alpha1"
	fakeghsourceclientset "knative.dev/eventing-github/pkg/client/clientset/versioned/fake"
	fakesourceclient "knative.dev/eventing-github/pkg/client/injection/client/fake"
	fakesourceinformer "knative.dev/eventing-github/pkg/client/injection/informers/sources/v1alpha1/githubsource/fake"
	cminformer "knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/system"
)

var ignoreTypeMeta = cmpopts.IgnoreFields(metav1.TypeMeta{}, "Kind", "APIVersion")

// initializeControllerAssets is a shared helper for controller initialization.
func initializeControllerAssets(t *testing.T, r test.Resources, grs []*v1alpha1.GitRepository, sources []ghsource.GitHubSource) (testAssets, func()) {
	t.Helper()
	ctx, _ := test.SetupFakeContext(t)
	ctx, cancel := context.WithCancel(ctx)
	// Set up all Tekton Pipelines and Triggers test data objects
	clients := test.SeedResources(t, ctx, r)

	// Set up all repos
	reposInformer := fakerepoinformer.Get(ctx)
	workflowsClient := fakeworkflowsclient.Get(ctx)
	for _, gr := range grs {
		gr := gr.DeepCopy()
		if err := reposInformer.Informer().GetIndexer().Add(gr); err != nil {
			t.Fatal(err)
		}
		if _, err := workflowsClient.WorkflowsV1alpha1().GitRepositories(gr.Namespace).Create(context.Background(), gr, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}

	// Set up all eventsources
	sourceInformer := fakesourceinformer.Get(ctx)
	sourceClient := fakesourceclient.Get(ctx)
	for _, s := range sources {
		s := s.DeepCopy()
		if err := sourceInformer.Informer().GetIndexer().Add(s); err != nil {
			t.Fatal(err)
		}
		if _, err := sourceClient.SourcesV1alpha1().GitHubSources(s.Namespace).Create(context.Background(), s, metav1.CreateOptions{}); err != nil {
			t.Fatal(err)
		}
	}

	configMapWatcher := cminformer.NewInformedWatcher(clients.Kube, system.Namespace())
	ctl := repos.NewController(ctx, configMapWatcher)
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
		GHSourceClient:  sourceClient,
	}, cancel
}

type reposTest struct {
	test.Resources `json:"inline"`
	Test           *testing.T
	TestAssets     testAssets
	Cancel         func()
}

type testAssets struct {
	test.Assets
	WorkflowsClient *fakeworkflowsclientset.Clientset
	GHSourceClient  *fakeghsourceclientset.Clientset
}

func newTest(r test.Resources, grs []*v1alpha1.GitRepository, sources []ghsource.GitHubSource, t *testing.T) *reposTest {
	t.Helper()
	testAssets, cancel := initializeControllerAssets(t, r, grs, sources)
	return &reposTest{
		Resources:  r,
		Test:       t,
		TestAssets: testAssets,
		Cancel:     cancel,
	}
}

func TestCreateGithubWebhook(t *testing.T) {
	tcs := []struct {
		name            string
		repo            *v1alpha1.GitRepository
		wantRepoStatus  v1alpha1.RepoStatus
		wantEventSource *ghsource.GitHubSource
	}{{
		name: "git-webhook connection",
		repo: parse.MustParseRepo(t, "myrepo", "default", `
spec:
  url: https://github.com/tektoncd/pipeline
  accessToken:
    name: knative-githubsecret
    key: accessToken
  webhookSecret:
    name: knative-webhooksecret
    key: secretToken
`),
		wantEventSource: parse.MustParseGitHubSource(t, `
metadata:
  name: myrepo
  namespace: default
  labels:
    workflows.tekton.dev/repo: myrepo
  ownerReferences:
  - apiVersion: workflows.tekton.dev/v1alpha1
    kind: GitRepository
    name: myrepo
    controller: true
    blockOwnerDeletion: true
spec:
  accessToken:
    secretKeyRef:
      name: knative-githubsecret
      key: accessToken
  secretToken:
    secretKeyRef:
      name: knative-webhooksecret
      key: secretToken
  ownerAndRepository: tektoncd/pipeline
  serviceAccountName: default
  eventTypes: [push]
  sink:
    ref:
      apiVersion: v1
      kind: Service
      name: el-workflows-listener
      namespace: tekton-workflows
`),
	}}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			prt := newTest(test.Resources{}, []*v1alpha1.GitRepository{tc.repo}, []ghsource.GitHubSource{}, t)
			defer prt.Cancel()

			c := prt.TestAssets.Controller
			reconcileError := c.Reconciler.Reconcile(context.Background(), fmt.Sprintf("%s/%s", tc.repo.Namespace, tc.repo.Name))
			if reconcileError != nil {
				t.Errorf("unexpected reconcile err %s", reconcileError)
			}
			var err error

			gotRepo, err := prt.TestAssets.WorkflowsClient.WorkflowsV1alpha1().GitRepositories(tc.repo.Namespace).Get(context.Background(), tc.repo.Name, metav1.GetOptions{})
			if err != nil {
				t.Fatalf("error getting repo: %s", err)
			}
			if d := cmp.Diff(tc.wantRepoStatus, gotRepo.Status); d != "" {
				t.Errorf("wrong repo status: %s", d)
			}

			gotSources, err := prt.TestAssets.GHSourceClient.SourcesV1alpha1().GitHubSources(tc.repo.Namespace).List(context.Background(), metav1.ListOptions{})
			if err != nil {
				t.Fatalf("error getting sources: %s", err)
			}
			if len(gotSources.Items) != 1 {
				t.Fatalf("unexpected number of github sources: %d", len(gotSources.Items))
			}
			gotSource := gotSources.Items[0]
			if d := cmp.Diff(*tc.wantEventSource, gotSource, ignoreTypeMeta); d != "" {
				t.Errorf("wrong eventsource: %s", d)
			}
		})
	}
}
