package workflows

import (
	"context"

	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
	workflowsclientset "github.com/tektoncd/experimental/workflows/pkg/client/clientset/versioned"
	workflowsreconciler "github.com/tektoncd/experimental/workflows/pkg/client/injection/reconciler/workflows/v1alpha1/workflow"
	"github.com/tektoncd/experimental/workflows/pkg/convert"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	listers "github.com/tektoncd/triggers/pkg/client/listers/triggers/v1beta1"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

type Reconciler struct {
	TriggerLister      listers.TriggerLister
	TriggerClientSet   triggersclientset.Interface
	WorkflowsClientSet workflowsclientset.Interface
}

var _ workflowsreconciler.Interface = (*Reconciler)(nil)

var ReasonCouldntGetRepo = "CouldntGetRepo"

func (r *Reconciler) ReconcileKind(ctx context.Context, w *v1alpha1.Workflow) reconciler.Event {
	repos, err := r.ReconcileRepos(ctx, w)
	if err != nil {
		return err
	}
	return r.ReconcileTriggers(ctx, w, repos)
}

func (r *Reconciler) ReconcileRepos(ctx context.Context, w *v1alpha1.Workflow) ([]*v1alpha1.GitRepository, reconciler.Event) {
	repoEvents := getEventsByRepo(w)
	var out []*v1alpha1.GitRepository
	for _, repo := range w.Spec.Repos {
		existing, err := r.WorkflowsClientSet.WorkflowsV1alpha1().GitRepositories(w.Namespace).Get(ctx, repo.Name, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				w.Status.MarkFailed(ReasonCouldntGetRepo, "Couldn't get repo %s/%s: %s", w.Namespace, repo.Name, err)
			}
			return nil, controller.NewPermanentError(err)
		}
		// Update repo spec to add events defined in the workflow triggers
		// TODO: Add a finalizer to remove these events if the workflow is deleted,
		// as long as no other workflows need these events
		existingEvents := sets.NewString(existing.Spec.EventTypes...)
		repoEvents := sets.NewString(repoEvents[repo.Name]...)
		wantEvents := repoEvents.Union(existingEvents)
		if !wantEvents.Equal(existingEvents) {
			newRepo := existing.DeepCopy()
			newRepo.Spec.EventTypes = wantEvents.List()
			_, err := r.WorkflowsClientSet.WorkflowsV1alpha1().GitRepositories(w.Namespace).Update(ctx, newRepo, metav1.UpdateOptions{})
			if err != nil {
				return nil, err
			}
		}
		out = append(out, existing)
	}
	return out, nil
}

func getEventsByRepo(w *v1alpha1.Workflow) map[string][]string {
	repoEvents := map[string][]string{}
	for _, t := range w.Spec.Triggers {
		if t.Event != nil && t.Event.Source.Repo != "" {
			repoName := t.Event.Source.Repo
			events, ok := repoEvents[repoName]
			if !ok {
				repoEvents[repoName] = v1alpha1.GetEventTypes(t.Event.Types)
			} else {
				repoEvents[repoName] = append(events, v1alpha1.GetEventTypes(t.Event.Types)...)
			}
		}
	}
	return repoEvents
}

func (r *Reconciler) ReconcileTriggers(ctx context.Context, w *v1alpha1.Workflow, grs []*v1alpha1.GitRepository) reconciler.Event {
	workflowTriggers, err := convert.ToTriggers(w, grs)
	if err != nil {
		return controller.NewPermanentError(err)
	}
	wantTriggers := buildMap(workflowTriggers)
	existingTriggers, err := r.TriggerLister.Triggers(w.Namespace).List(k8slabels.SelectorFromSet(map[string]string{v1alpha1.WorkflowLabelKey: w.Name}))
	if err != nil {
		return err
	}
	gotTriggers := buildMap(existingTriggers)
	var ops []triggerOp
	for name, t := range wantTriggers {
		got, ok := gotTriggers[name]
		if !ok {
			ops = append(ops, triggerOp{trigger: t, op: v1.Create})
		} else if !equal(t, got) {
			new := got.DeepCopy()
			new.Spec = t.Spec
			new.Labels = t.Labels
			ops = append(ops, triggerOp{trigger: new, op: v1.Update})
		}
	}
	for name, t := range gotTriggers {
		_, ok := wantTriggers[name]
		if !ok {
			ops = append(ops, triggerOp{trigger: t, op: v1.Delete})
		}
	}
	return r.updateTriggers(ctx, ops, w.Namespace)
}

type triggerOp struct {
	trigger *v1beta1.Trigger
	op      v1.OperationType
}

func buildMap(ts []*v1beta1.Trigger) map[string]*v1beta1.Trigger {
	if len(ts) == 0 {
		return nil
	}
	out := make(map[string]*v1beta1.Trigger)
	for _, t := range ts {
		out[t.Name] = t
	}
	return out
}

func equal(x, y *v1beta1.Trigger) bool {
	return equality.Semantic.DeepEqual(x.Spec, y.Spec) && equality.Semantic.DeepEqual(x.Labels, y.Labels)
}

func (r *Reconciler) updateTriggers(ctx context.Context, ts []triggerOp, namespace string) error {
	logger := logging.FromContext(ctx)
	g := new(errgroup.Group)
	for _, t := range ts {
		t := t // https://go.dev/doc/faq#closures_and_goroutines
		g.Go(func() error {
			logger.Infof("Performing operation %s on Trigger %s in namespace %s", t.op, t.trigger.Name, namespace)
			var err error
			switch t.op {
			case v1.Create:
				_, err = r.TriggerClientSet.TriggersV1beta1().Triggers(namespace).Create(ctx, t.trigger, metav1.CreateOptions{})
			case v1.Update:
				_, err = r.TriggerClientSet.TriggersV1beta1().Triggers(namespace).Update(ctx, t.trigger, metav1.UpdateOptions{})
			case v1.Delete:
				err = r.TriggerClientSet.TriggersV1beta1().Triggers(namespace).Delete(ctx, t.trigger.Name, metav1.DeleteOptions{})
			}
			return err
		})
	}
	return g.Wait()
}
