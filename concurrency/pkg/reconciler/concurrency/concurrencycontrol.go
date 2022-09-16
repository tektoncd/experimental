package concurrency

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	pipelinerunreconciler "github.com/tektoncd/pipeline/pkg/client/injection/reconciler/pipeline/v1beta1/pipelinerun"
	"github.com/tektoncd/pipeline/pkg/substitution"

	"github.com/tektoncd/experimental/concurrency/pkg/apis/concurrency/v1alpha1"
	listersv1alpha1 "github.com/tektoncd/experimental/concurrency/pkg/client/listers/concurrency/v1alpha1"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"

	"gomodules.xyz/jsonpatch/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8slabels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/kubernetes"
	"knative.dev/pkg/logging"
	pkgreconciler "knative.dev/pkg/reconciler"
)

// Reconciler implements controller.Reconciler for Configuration resources.
type Reconciler struct {
	KubeClientSet            kubernetes.Interface
	PipelineClientSet        clientset.Interface
	Clock                    clock.PassiveClock
	ConcurrencyControlLister listersv1alpha1.ConcurrencyControlLister
	PipelineRunLister        listers.PipelineRunLister
}

var (
	// Check that our Reconciler implements pipelinerunreconciler.Interface
	_ pipelinerunreconciler.Interface = (*Reconciler)(nil)

	cancelPipelineRunPatchBytes []byte

	paramPatterns = []string{
		"params.%s",
		"params[%q]",
		"params['%s']",
	}
)

const (
	// objectIndividualVariablePattern is the reference pattern for object individual keys params.<object_param_name>.<key_name>
	objectIndividualVariablePattern = "params.%s.%s"
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

// ReconcileKind docstring
func (r *Reconciler) ReconcileKind(ctx context.Context, pr *v1beta1.PipelineRun) pkgreconciler.Event {
	logger := logging.FromContext(ctx)
	if hasConcurrencyLabels(pr.ObjectMeta) {
		// Assume concurrency controls have already been handled in a previous reconcile loop
		// If pipelinerun is pending here, it's an error
		logger.Infof("pr %s in ns %s already has concurrency labels: %s, skipping", pr.Name, pr.Namespace, pr.Labels)
		return nil
	}

	// Find all concurrency controls in the namespace and determine which ones match
	labelSelector := k8slabels.SelectorFromSet(make(map[string]string))
	ccs, err := r.ConcurrencyControlLister.ConcurrencyControls(pr.Namespace).List(labelSelector)
	if err != nil {
		return err
	}
	logger.Infof("found %d concurrency controls in ns %s", len(ccs), pr.Namespace)
	labelsToAdd := make(map[string]string)
	for _, cc := range ccs {
		if cc.Spec.Strategy != "Cancel" {
			return fmt.Errorf("unsupported concurrency strategy for control %s: %s. Only 'Cancel' is supported", cc.ObjectMeta.Name, cc.Spec.Strategy)
		}
		if matchesLabelSelector(pr.ObjectMeta, cc.Spec.Selector) {
			// Label key can contain only one slash, separating prefix (tekton.dev) from the rest of the key
			k := fmt.Sprintf("tekton.dev/concurrency-%s", cc.ObjectMeta.Name)
			v := getConcurrencyKey(ctx, pr, cc)
			labelsToAdd[k] = v
			logger.Infof("found matching concurrency control %s for PR %s", cc.Name, pr.Name)
		}
	}

	// Find other PRs in the same concurrency group
	// Labels are treated as "or", not "and"; i.e. a PR is canceled if any of its concurrency keys match
	var prsToCancel []*v1beta1.PipelineRun
	for k, v := range labelsToAdd {
		prs, err := r.PipelineRunLister.PipelineRuns(pr.Namespace).List(k8slabels.SelectorFromSet(map[string]string{k: v}))
		if err != nil {
			return err
		}
		prsToCancel = append(prsToCancel, prs...)
	}

	wg := sync.WaitGroup{}

	for _, otherPR := range prsToCancel {
		if !otherPR.IsDone() {
			wg.Add(1)
			go func(p *v1beta1.PipelineRun) {
				defer wg.Done()
				logger.Infof("canceling PR %s", p.Name)
				_ = r.cancelOtherPipelineRunViaAPI(ctx, p.Namespace, p.Name)
			}(otherPR)
		}
	}
	// TODO: For actual implementation this shouldn't block
	wg.Wait()

	// If no matching concurrency controls, apply label None
	if len(labelsToAdd) == 0 {
		labelsToAdd = map[string]string{"tekton.dev/concurrency": "None"}

		logger.Infof("no matching concurrency controls for PR %s", pr.Name)
	}
	logger.Infof("starting PR %s", pr.Name)
	// Note: we can't really distinguish between a PR that a user wants to be pending, and a PR that is pending only to avoid concurrent execution.
	// We start all of them.
	return r.updateLabelsAndStartPR(ctx, pr, labelsToAdd)
}

func (r *Reconciler) cancelOtherPipelineRunViaAPI(ctx context.Context, namespace, name string) error {
	_, err := r.PipelineClientSet.TektonV1beta1().PipelineRuns(namespace).Patch(ctx, name, types.JSONPatchType, cancelPipelineRunPatchBytes, metav1.PatchOptions{})
	return err
}

func (r *Reconciler) updateLabelsAndStartPR(ctx context.Context, pr *v1beta1.PipelineRun, labels map[string]string) error {
	logger := logging.FromContext(ctx)
	newPr, err := r.PipelineRunLister.PipelineRuns(pr.Namespace).Get(pr.Name)
	if err != nil {
		return fmt.Errorf("error getting PipelineRun %s when updating labels/annotations: %w", pr.Name, err)
	}
	logger.Infof("pr %s in ns %s had status %s and labels %s. clearing status and adding labels %s", pr.Name, pr.Namespace,
		newPr.Spec.Status, newPr.Labels, labels)
	newPr = newPr.DeepCopy()
	newPr.Labels = pr.Labels
	addLabels(newPr, labels)
	newPr.Spec.Status = ""
	_, err = r.PipelineClientSet.TektonV1beta1().PipelineRuns(pr.Namespace).Update(ctx, newPr, metav1.UpdateOptions{})

	return err
}

// Returns true if any of an object's labels represent a concurrency control
func hasConcurrencyLabels(om metav1.ObjectMeta) bool {
	for k := range om.Labels {
		if strings.HasPrefix(k, "tekton.dev/concurrency") {
			return true
		}
	}
	return false
}

func getConcurrencyKey(ctx context.Context, pr *v1beta1.PipelineRun, c *v1alpha1.ConcurrencyControl) string {
	stringReplacements := map[string]string{}

	// Set all the default stringReplacements
	for _, p := range c.Spec.Params {
		if p.Default != nil {
			switch p.Default.Type {
			case v1beta1.ParamTypeArray:
				for _, pattern := range paramPatterns {
					for i := 0; i < len(p.Default.ArrayVal); i++ {
						stringReplacements[fmt.Sprintf(pattern+"[%d]", p.Name, i)] = p.Default.ArrayVal[i]
					}
				}
			case v1beta1.ParamTypeObject:
				for k, v := range p.Default.ObjectVal {
					stringReplacements[fmt.Sprintf(objectIndividualVariablePattern, p.Name, k)] = v
				}
			default:
				for _, pattern := range paramPatterns {
					stringReplacements[fmt.Sprintf(pattern, p.Name)] = p.Default.StringVal
				}
			}
		}
	}
	// Set and overwrite params with the ones from the PipelineRun
	prStrings := paramsFromPipelineRun(ctx, pr)

	for k, v := range prStrings {
		stringReplacements[k] = v
	}
	key := c.Spec.Key

	return substitution.ApplyReplacements(key, stringReplacements)
}

func paramsFromPipelineRun(ctx context.Context, pr *v1beta1.PipelineRun) map[string]string {
	// stringReplacements is used for standard single-string stringReplacements,
	// while arrayReplacements/objectReplacements contains arrays/objects that need to be further processed.
	stringReplacements := map[string]string{}
	for _, p := range pr.Spec.Params {
		switch p.Value.Type {
		case v1beta1.ParamTypeArray:
			for _, pattern := range paramPatterns {
				// array indexing for param is alpha feature
				for i := 0; i < len(p.Value.ArrayVal); i++ {
					stringReplacements[fmt.Sprintf(pattern+"[%d]", p.Name, i)] = p.Value.ArrayVal[i]
				}

			}
		case v1beta1.ParamTypeObject:
			for k, v := range p.Value.ObjectVal {
				stringReplacements[fmt.Sprintf(objectIndividualVariablePattern, p.Name, k)] = v
			}
		default:
			for _, pattern := range paramPatterns {
				stringReplacements[fmt.Sprintf(pattern, p.Name)] = p.Value.StringVal
			}
		}
	}
	return stringReplacements
}

func matchesLabelSelector(om metav1.ObjectMeta, selector metav1.LabelSelector) bool {
	for k, v := range selector.MatchLabels {
		existing, ok := om.Labels[k]
		if !ok || v != existing {
			return false
		}
	}
	return true
}

func addLabels(pr *v1beta1.PipelineRun, labels map[string]string) {
	if len(pr.ObjectMeta.Labels) == 0 {
		pr.ObjectMeta.Labels = make(map[string]string)
	}
	for k, v := range labels {
		pr.ObjectMeta.Labels[k] = v
	}
}
