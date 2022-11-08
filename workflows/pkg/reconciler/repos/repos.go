package repos

import (
	"context"
	"reflect"
	"strings"

	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
	reposreconciler "github.com/tektoncd/experimental/workflows/pkg/client/injection/reconciler/workflows/v1alpha1/gitrepository"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ghsource "knative.dev/eventing-github/pkg/apis/sources/v1alpha1"
	ghsourceclient "knative.dev/eventing-github/pkg/client/clientset/versioned"
	ghsourcelister "knative.dev/eventing-github/pkg/client/listers/sources/v1alpha1"
	duckv1 "knative.dev/pkg/apis/duck/v1"
	"knative.dev/pkg/kmeta"
	"knative.dev/pkg/reconciler"
)

type Reconciler struct {
	GithubSourceLister    ghsourcelister.GitHubSourceLister
	GitHubSourceClientSet ghsourceclient.Interface
}

var _ reposreconciler.Interface = (*Reconciler)(nil)

const repoLabelKey = "workflows.tekton.dev/repo"

func (r *Reconciler) ReconcileKind(ctx context.Context, gr *v1alpha1.GitRepository) reconciler.Event {
	want := toGitHubSource(gr)
	existing, err := r.GithubSourceLister.GitHubSources(gr.Namespace).Get(gr.Name)
	var s *ghsource.GitHubSource
	if err != nil {
		if k8serrors.IsNotFound(err) {
			s, err = r.GitHubSourceClientSet.SourcesV1alpha1().GitHubSources(want.Namespace).Create(ctx, want, metav1.CreateOptions{})

		} else {
			return err
		}
	} else {
		if !reflect.DeepEqual(existing.Spec, want.Spec) {
			s, err = r.GitHubSourceClientSet.SourcesV1alpha1().GitHubSources(want.Namespace).Update(ctx, want, metav1.UpdateOptions{})
		} else {
			s = existing
		}
	}
	if err != nil {
		return err
	}
	gr.Status.Conditions = s.Status.Conditions
	return nil
}

func toGitHubSource(gr *v1alpha1.GitRepository) *ghsource.GitHubSource {
	eventTypes := gr.Spec.EventTypes
	if len(eventTypes) == 0 {
		// Knative Github eventsource requires at least one event type, so we will create it with an arbitrary one
		// until it is updated by the workflows controller
		eventTypes = []string{"push"}
	}
	return &ghsource.GitHubSource{
		Spec: ghsource.GitHubSourceSpec{
			OwnerAndRepository: ParseOwnerAndRepo(gr.Spec.URL),
			EventTypes:         eventTypes,
			ServiceAccountName: "default",
			AccessToken: ghsource.SecretValueFromSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: gr.Spec.AccessToken.Name},
					Key:                  gr.Spec.AccessToken.Key,
				}},
			SecretToken: ghsource.SecretValueFromSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{Name: gr.Spec.WebhookSecret.Name},
					Key:                  gr.Spec.WebhookSecret.Key,
				}},
			SourceSpec: duckv1.SourceSpec{Sink: duckv1.Destination{
				Ref: &duckv1.KReference{Kind: "Service", APIVersion: "v1", Namespace: v1alpha1.WorkflowsNamespace, Name: "el-workflows-listener"}}},
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:            gr.Name,
			Namespace:       gr.Namespace,
			Labels:          map[string]string{repoLabelKey: gr.Name},
			OwnerReferences: []metav1.OwnerReference{*kmeta.NewControllerRef(gr)},
		}}
}

func ParseOwnerAndRepo(url string) string {
	s := strings.TrimPrefix(url, "https://github.com/")
	s = strings.TrimPrefix(s, "http://github.com/")
	return s
}
