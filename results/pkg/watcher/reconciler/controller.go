package reconciler

import (
	"context"
	"encoding/json"

	"github.com/tektoncd/experimental/results/pkg/watcher/convert"
	pb "github.com/tektoncd/experimental/results/proto/proto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	taskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/taskrun"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"go.uber.org/zap"
	jsonpatch "gomodules.xyz/jsonpatch/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	_ "knative.dev/pkg/system/testing"
)

const (
	path   = "/metadata/annotations/results.tekton.dev~1id"
	idName = "results.tekton.dev/id"
)

// NewController creates a Controller with provided context and configmap
func NewController(ctx context.Context, cmw configmap.Watcher, client pb.ResultsClient) *controller.Impl {
	logger := logging.FromContext(ctx)
	taskRunInformer := taskruninformer.Get(ctx)
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
	return impl
}

type reconciler struct {
	logger            *zap.SugaredLogger
	client            pb.ResultsClient
	taskRunLister     listers.TaskRunLister
	pipelineclientset versioned.Interface
}

func (r *reconciler) Reconcile(ctx context.Context, key string) error {
	r.logger.Infof("reconciling resource key: %s", key)
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		r.logger.Errorf("invalid resource key: %s", key)
		return nil
	}

	// Get the Task Run resource with this namespace/name
	tr, err := r.pipelineclientset.TektonV1beta1().TaskRuns(namespace).Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		// The resource no longer exists, in which case we stop processing.
		r.logger.Infof("task run %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		r.logger.Errorf("Error retrieving Result %q: %s", name, err)
		return err
	}

	r.logger.Infof("Receiving new Result %s/%s", namespace, tr.Name)

	// Send the new status of the Result to the API server.
	p, err := convert.ToProto(tr)
	if err != nil {
		r.logger.Errorf("Error converting to proto: %v", err)
		return err
	}
	res := &pb.Result{
		Executions: []*pb.Execution{{
			Execution: &pb.Execution_TaskRun{p},
		}},
	}

	// Create a Result if it does not exist in results server, update existing one otherwise.
	if val, ok := p.GetMetadata().GetAnnotations()[idName]; ok {
		res.Name = val
		if _, err := r.client.UpdateResult(ctx, &pb.UpdateResultRequest{
			Name:   val,
			Result: res,
		}); err != nil {
			r.logger.Errorf("Error updating TaskRun %s: %v", name, err)
			return err
		}
		r.logger.Infof("Sending updates for TaskRun %s/%s (result: %s)", namespace, tr.Name, val)
	} else {
		res, err = r.client.CreateResult(ctx, &pb.CreateResultRequest{
			Result: res,
		})
		if err != nil {
			r.logger.Errorf("Error creating Result %s: %v", name, err)
			return err
		}
		path, err := annotationPath(res.GetName(), path, "add")
		if err != nil {
			r.logger.Errorf("Error jsonpatch for Result %s : %v", name, err)
			return err
		}
		r.pipelineclientset.TektonV1beta1().TaskRuns(namespace).Patch(name, types.JSONPatchType, path)
		r.logger.Infof("Creating a new TaskRun result %s/%s (result: %s)", namespace, tr.Name, res.GetName())
	}
	return nil
}

// AnnotationPath creates a jsonpatch path used for adding results_id to Result annotations field.
func annotationPath(val string, path string, op string) ([]byte, error) {
	patches := []jsonpatch.JsonPatchOperation{{
		Operation: op,
		Path:      path,
		Value:     val,
	}}
	return json.Marshal(patches)
}
