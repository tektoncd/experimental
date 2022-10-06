package workflows

import (
	"context"

	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
	workflowsreconciler "github.com/tektoncd/experimental/workflows/pkg/client/injection/reconciler/workflows/v1alpha1/workflow"
	"github.com/tektoncd/experimental/workflows/pkg/convert"
	"github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	triggersclientset "github.com/tektoncd/triggers/pkg/client/clientset/versioned"
	listers "github.com/tektoncd/triggers/pkg/client/listers/triggers/v1beta1"
	"golang.org/x/sync/errgroup"
	v1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

type Reconciler struct {
	TriggerLister    listers.TriggerLister
	TriggerClientSet triggersclientset.Interface
}

var _ workflowsreconciler.Interface = (*Reconciler)(nil)

func (r *Reconciler) ReconcileKind(ctx context.Context, w *v1alpha1.Workflow) reconciler.Event {
	workflowTriggers, err := convert.ToTriggers(w)
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
			ops = append(ops, triggerOp{trigger: t, op: v1.Update})
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
	return equality.Semantic.DeepEqual(x.Spec, y.Spec) &&
		equality.Semantic.DeepEqual(x.Labels, y.Labels) &&
		equality.Semantic.DeepEqual(x.Annotations, y.Annotations)
}

func (r *Reconciler) updateTriggers(ctx context.Context, ts []triggerOp, namespace string) error {
	logger := logging.FromContext(ctx)
	g := new(errgroup.Group)
	for _, t := range ts {
		t := t // https://go.dev/doc/faq#closures_and_goroutines
		g.Go(func() error {
			logger.Infof("updating Trigger %s in namespace %s", t.trigger.Name, namespace)
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
