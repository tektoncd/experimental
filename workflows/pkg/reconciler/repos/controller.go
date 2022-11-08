package repos

import (
	"context"

	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
	reposinformer "github.com/tektoncd/experimental/workflows/pkg/client/injection/informers/workflows/v1alpha1/gitrepository"
	reposreconciler "github.com/tektoncd/experimental/workflows/pkg/client/injection/reconciler/workflows/v1alpha1/gitrepository"
	"k8s.io/client-go/tools/cache"
	ghsourceclient "knative.dev/eventing-github/pkg/client/injection/client"
	ghsourceinformer "knative.dev/eventing-github/pkg/client/injection/informers/sources/v1alpha1/githubsource"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
)

// NewController creates a Reconciler and returns the result of NewImpl.
func NewController(
	ctx context.Context,
	cmw configmap.Watcher,
) *controller.Impl {
	reposInformer := reposinformer.Get(ctx)
	ghSourceInformer := ghsourceinformer.Get(ctx)
	r := &Reconciler{
		GithubSourceLister:    ghSourceInformer.Lister(),
		GitHubSourceClientSet: ghsourceclient.Get(ctx),
	}
	impl := reposreconciler.NewImpl(ctx, r)
	reposInformer.Informer().AddEventHandler(controller.HandleAll(impl.Enqueue))
	ghSourceInformer.Informer().AddEventHandler(cache.FilteringResourceEventHandler{
		FilterFunc: controller.FilterController(&v1alpha1.GitRepository{}),
		Handler:    controller.HandleAll(impl.EnqueueControllerOf),
	})
	return impl
}
