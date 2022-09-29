package concurrency

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/tektoncd/experimental/concurrency/pkg/apis/concurrency/v1alpha1"
	listersv1alpha1 "github.com/tektoncd/experimental/concurrency/pkg/client/listers/concurrency/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"golang.org/x/sync/errgroup"
	"gomodules.xyz/jsonpatch/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	logging "knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

// Reconciler implements controller.Reconciler
type Reconciler struct {
	ConcurrencyControlLister listersv1alpha1.ConcurrencyControlLister
	PipelineClientSet        clientset.Interface
	PipelineRunLister        listers.PipelineRunLister
}

var (
	cancelPipelineRunPatchBytes     []byte
	concurrencyControlsAppliedLabel = "tekton.dev/concurrency"
)

func init() {
	var err error
	cancelPipelineRunPatchBytes, err = json.Marshal([]jsonpatch.JsonPatchOperation{
		{
			Operation: "add",
			Path:      "/spec/status",
			Value:     v1beta1.PipelineRunSpecStatusCancelled,
		}})
	if err != nil {
		log.Fatalf("failed to marshal PipelineRun cancel patch bytes: %v", err)
	}
}

// ReconcileKind reconciles PipelineRuns
func (r *Reconciler) ReconcileKind(ctx context.Context, pr *v1beta1.PipelineRun) pkgreconciler.Event {
	logger := logging.FromContext(ctx)
	if !pr.IsPending() || concurrencyControlsPreviouslyApplied(pr) {
		return nil
	}

	// Find all concurrency controls in the namespace and determine which ones match
	ccs, err := r.ConcurrencyControlLister.ConcurrencyControls(pr.Namespace).List(k8slabels.Everything())
	if err != nil {
		return err
	}
	logger.Infof("found %d concurrency controls in namespace %s", len(ccs), pr.Namespace)

	prsToCancel := sets.NewString()
	for _, cc := range ccs {
		if !matches(pr, cc) {
			// Concurrency control does not apply to this PipelineRun
			continue
		}
		// If concurrency control matches the current pipelinerun, get all pipelineruns matching the same label selector.
		// Cancel them all except the one currently running.
		matchingPRs, err := r.PipelineRunLister.PipelineRuns(pr.Namespace).List(k8slabels.SelectorFromSet(cc.Spec.Selector.MatchLabels))
		if err != nil {
			return err
		}
		for _, matchingPR := range matchingPRs {
			if matchingPR.Name != pr.Name && !matchingPR.IsDone() {
				prsToCancel.Insert(matchingPR.Name)
			}
		}
	}
	err = r.cancelPipelineRuns(ctx, pr.Namespace, prsToCancel.List())
	if err != nil {
		return fmt.Errorf("error canceling PipelineRuns in the same concurrency group as %s: %s", pr.Name, err)
	}

	return r.updateLabelsAndStartPipelineRun(ctx, pr)
}

func matches(pr *v1beta1.PipelineRun, cc *v1alpha1.ConcurrencyControl) bool {
	return k8slabels.SelectorFromSet(cc.Spec.Selector.MatchLabels).Matches(k8slabels.Set(pr.Labels))
}

// concurrencyControlsPreviouslyApplied returns true if concurrency controls have been applied in a previous reconcile loop,
// and no further work is necessary
func concurrencyControlsPreviouslyApplied(pr *v1beta1.PipelineRun) bool {
	_, ok := pr.Labels[concurrencyControlsAppliedLabel]
	return ok
}

func (r *Reconciler) cancelPipelineRuns(ctx context.Context, namespace string, names []string) error {
	logger := logging.FromContext(ctx)
	g := new(errgroup.Group)
	for _, n := range names {
		n := n // https://go.dev/doc/faq#closures_and_goroutines
		g.Go(func() error {
			logger.Infof("canceling PipelineRun %s in namespace %s", n, namespace)
			return r.cancelPipelineRun(ctx, namespace, n)
		})
	}
	// TODO: We may want to implement a solution that avoids blocking until all PipelineRuns have been canceled.
	// However, this is probably good enough for the time being.
	// This is similar to how the PipelineRun reconciler cancels child TaskRuns
	// (see https://github.com/tektoncd/pipeline/blob/main/pkg/reconciler/pipelinerun/cancel.go).
	return g.Wait()
}

func (r *Reconciler) cancelPipelineRun(ctx context.Context, namespace, name string) error {
	// TODO: Add support for graceful cancellation and graceful stopping
	_, err := r.PipelineClientSet.TektonV1beta1().PipelineRuns(namespace).Patch(ctx, name, types.JSONPatchType, cancelPipelineRunPatchBytes, metav1.PatchOptions{})
	if errors.IsNotFound(err) {
		// The PipelineRun may have been deleted in the meantime
		return nil
	} else if err != nil {
		return fmt.Errorf("error canceling PipelineRun %s: %s", name, err)
	}
	return nil
}

// updateLabelsAndStartPipelineRun marks the PipelineRun with a label indicating concurrency controls have been applied.
// If it was modified to be pending by the mutating admission webhook (rather than started as pending by the user),
// it starts the PipelineRun.
func (r *Reconciler) updateLabelsAndStartPipelineRun(ctx context.Context, pr *v1beta1.PipelineRun) error {
	newPR, err := r.PipelineRunLister.PipelineRuns(pr.Namespace).Get(pr.Name)
	if err != nil {
		return fmt.Errorf("error getting PipelineRun %s in namespace %s when updating labels: %w", pr.Name, pr.Namespace, err)
	}
	newPR = newPR.DeepCopy()
	newPR.Labels = pr.Labels
	if len(newPR.ObjectMeta.Labels) == 0 {
		newPR.ObjectMeta.Labels = make(map[string]string)
	}
	newPR.Labels[concurrencyControlsAppliedLabel] = "true"
	if _, ok := newPR.Labels[v1alpha1.LabelToStartPR]; ok {
		delete(newPR.Labels, v1alpha1.LabelToStartPR)
		// This PipelineRun was marked as pending by the mutating admission webhook, not the user. OK to start it.
		newPR.Spec.Status = ""
	}
	_, err = r.PipelineClientSet.TektonV1beta1().PipelineRuns(pr.Namespace).Update(ctx, newPR, metav1.UpdateOptions{})
	return err
}
