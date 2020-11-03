package reconciler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/tektoncd/experimental/results/pkg/watcher/convert"
	pb "github.com/tektoncd/experimental/results/proto/proto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	pipelineruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/pipelinerun"
	taskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/taskrun"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"go.uber.org/zap"
	jsonpatch "gomodules.xyz/jsonpatch/v2"
	"k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
)

const (
	Path   = "/metadata/annotations/results.tekton.dev~1id"
	IDName = "results.tekton.dev/id"
)

// NewController creates a Controller with provided context and configmap
func NewController(ctx context.Context, cmw configmap.Watcher, client pb.ResultsClient) *controller.Impl {
	logger := logging.FromContext(ctx)
	taskRunInformer := taskruninformer.Get(ctx)
	pipelineRunInformer := pipelineruninformer.Get(ctx)
	pipelineclientset := pipelineclient.Get(ctx)
	c := &reconciler{
		logger:            logger,
		client:            client,
		pipelineclientset: pipelineclientset,
	}
	impl := controller.NewImpl(c, c.logger, pipeline.PipelineRunControllerName)

	taskRunInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.Enqueue,
		UpdateFunc: controller.PassNew(impl.Enqueue),
	})
	pipelineRunInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc:    impl.Enqueue,
		UpdateFunc: controller.PassNew(impl.Enqueue),
	})
	return impl
}

type reconciler struct {
	logger            *zap.SugaredLogger
	client            pb.ResultsClient
	taskRunLister     listers.TaskRunLister
	pipelineRunLister listers.PipelineRunLister
	pipelineclientset versioned.Interface
}

func (r *reconciler) Reconcile(ctx context.Context, key string) error {
	r.logger.Infof("reconciling resource key: %s", key)
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		r.logger.Errorf("invalid resource key: %s", key)
		return err
	}

	pEMsg, prErr := r.reconcilePipelineRun(ctx, namespace, name)
	tEMsg, trErr := r.reconcileTaskRun(ctx, namespace, name)

	emsgs := []string{pEMsg, tEMsg}
	emsg, err := getReconcileError(emsgs, prErr, trErr)
	if err != nil {
		r.logger.With(zap.String("namespace", namespace), zap.String("name", name)).Error(emsg)
	}
	return err
}

func isReconcileSuccess(errs ...error) bool {
	for _, err := range errs {
		if err == nil {
			return true
		}
	}
	return false
}

func getReconcileError(emsgs []string, errs ...error) (string, error) {
	if isReconcileSuccess(errs...) {
		return "", nil
	}
	for idx, err := range errs {
		if !errors.IsNotFound(err) {
			return emsgs[idx], err
		}
	}
	return "Can't find valid Tekton Result resource", errors.NewNotFound(v1beta1.Resource("Tekton Result"), "Tekton Result")
}

func (r *reconciler) reconcileTaskRun(ctx context.Context, namespace string, name string) (emsg string, e error) {
	log := r.logger.With(zap.String("type", "TaskRun"), zap.String("namespace", namespace), zap.String("name", name))

	// Get the Task Run resource with this namespace/name
	taskRun, err := r.pipelineclientset.TektonV1beta1().TaskRuns(namespace).Get(name, metav1.GetOptions{})

	if errors.IsNotFound(err) {
		// The resource no longer exists, in which case we stop processing.
		return fmt.Sprintf("TaskRun %q/%q in work queue no longer exists", namespace, name), err
	} else if err != nil {
		return fmt.Sprintf("Error retrieving Result %q/%q", namespace, name), err
	}

	log.Info("Recieving new Result")
	// Send the new status of the Result to the API server.
	taskrunProto, err := convert.ToTaskRunProto(taskRun)
	if err != nil {
		return "Error converting to proto", err
	}

	trResult := &pb.Result{
		Executions: []*pb.Execution{{
			Execution: &pb.Execution_TaskRun{taskrunProto},
		}},
	}

	// Create a Result if it does not exist in results server, update existing one otherwise.
	if resultID, ok := taskrunProto.GetMetadata().GetAnnotations()[IDName]; ok {
		trResult.Name = resultID
		if _, err := r.client.UpdateResult(ctx, &pb.UpdateResultRequest{
			Name:   resultID,
			Result: trResult,
		}); err != nil {
			return fmt.Sprintf("Error updating TaskRun %q/%q", namespace, name), err
		}
		log.Infof("Sending updates, result id: %q", trResult.GetName())
	} else {
		if trResult, err = r.client.CreateResult(ctx, &pb.CreateResultRequest{
			Result: trResult,
		}); err != nil {
			return fmt.Sprintf("Error creating TaskRun Result %s/%q", namespace, name), err
		}
		path, err := AnnotationPath(trResult.GetName(), Path, "add")
		if err != nil {
			return fmt.Sprintf("Error jsonpatch for TaskRun Result %q/%q", name, err), err
		}
		r.pipelineclientset.TektonV1beta1().TaskRuns(namespace).Patch(name, types.JSONPatchType, path)
		log.Infof("Creating a new result: %s", namespace, taskRun.Name, trResult.GetName())
	}
	return "", nil
}

func (r *reconciler) reconcilePipelineRun(ctx context.Context, namespace string, name string) (emsg string, e error) {
	log := r.logger.With(zap.String("type", "PipelineRun"), zap.String("namespace", namespace), zap.String("name", name))

	// Get the Pipeline Run resource with this namespace/name
	pipelineRun, err := r.pipelineclientset.TektonV1beta1().PipelineRuns(namespace).Get(name, metav1.GetOptions{})

	if errors.IsNotFound(err) {
		// The resource no longer exists, in which case we stop processing.
		return fmt.Sprintf("PipelineRun %q/%q in work queue no longer exists", namespace, name), err
	} else if err != nil {
		return fmt.Sprintf("Error retrieving PipelineRun %q/%q", namespace, name), err
	}

	log.Info("Recieving new Result")
	// Send the new status of the Result to the API server.
	prProto, err := convert.ToPipelineRunProto(pipelineRun)
	if err != nil {
		return "Error converting PipelineRun to proto", err
	}

	prResult := &pb.Result{
		Executions: []*pb.Execution{{
			Execution: &pb.Execution_PipelineRun{prProto},
		}},
	}

	// Create a Result if it does not exist in results server, update existing one otherwise.
	if resultID, ok := prProto.GetMetadata().GetAnnotations()[IDName]; ok {
		prResult.Name = resultID
		if _, err := r.client.UpdateResult(ctx, &pb.UpdateResultRequest{
			Name:   resultID,
			Result: prResult,
		}); err != nil {
			return fmt.Sprintf("Error updating PipelineRun %q/%q", namespace, name), err
		}
		log.Infof("Sending updates, result id: %q", prResult.GetName())
	} else {
		if prResult, err = r.client.CreateResult(ctx, &pb.CreateResultRequest{
			Result: prResult,
		}); err != nil {
			return fmt.Sprintf("Error creating PipelineRun Result %q/%q", namespace, name), err
		}
		path, err := AnnotationPath(prResult.GetName(), Path, "add")
		if err != nil {
			return fmt.Sprintf("Error jsonpatch for PipelineRun Result %q/%q", namespace, name), err
		}
		r.pipelineclientset.TektonV1beta1().PipelineRuns(namespace).Patch(name, types.JSONPatchType, path)
		log.Infof("Creating a new result: %s", namespace, pipelineRun.Name, prResult.GetName())
	}
	return "", nil
}

// AnnotationPath creates a jsonpatch path used for adding results_id to Result
// annotations field.
func AnnotationPath(resultID string, path string, op string) ([]byte, error) {
	patches := []jsonpatch.JsonPatchOperation{{
		Operation: op,
		Path:      path,
		Value:     resultID,
	}}
	return json.Marshal(patches)
}
