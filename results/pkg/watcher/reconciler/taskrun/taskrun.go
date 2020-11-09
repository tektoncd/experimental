package taskrun

import (
	"context"
	"encoding/json"

	"github.com/tektoncd/experimental/results/pkg/watcher/convert"
	pb "github.com/tektoncd/experimental/results/proto/proto"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"go.uber.org/zap"
	"gomodules.xyz/jsonpatch/v2"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/logging"
)

const (
	path   = "/metadata/annotations/results.tekton.dev~1id"
	idName = "results.tekton.dev/id"
)

type Reconciler struct {
	logger            *zap.SugaredLogger
	client            pb.ResultsClient
	taskRunLister     listers.TaskRunLister
	pipelineclientset versioned.Interface
}

func (r *Reconciler) Reconcile(ctx context.Context, key string) error {
	logger := logging.FromContext(ctx)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		logger.Errorf("Invalid resource key: %s", key)
		return nil
	}
	logger.With(zap.String("Namespace", namespace), zap.String("Name", name))

	tr, err := r.pipelineclientset.TektonV1beta1().TaskRuns(namespace).Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		logger.Errorf("TaskRun in work queue no longer exists: %v", err)
		return nil
	}
	if err != nil {
		logger.Errorf("Error retrieving TaskRun: %v", err)
		return err
	}

	logger.Info("Recieving new TaskRun")
	trProto, err := convert.ToTaskRunProto(tr)
	if err != nil {
		logger.Errorf("Error converting TaskRun to its corresponding proto: %v", err)
		return err
	}

	trResult := &pb.Result{
		Executions: []*pb.Execution{{
			Execution: &pb.Execution_TaskRun{trProto},
		}},
	}

	if resultID, ok := trProto.GetMetadata().GetAnnotations()[idName]; ok {
		trResult.Name = resultID
		if _, err = r.client.UpdateResult(ctx, &pb.UpdateResultRequest{
			Name:   resultID,
			Result: trResult,
		}); err != nil {
			logger.Errorf("Error updating TaskRun: %v", err)
			return err
		}
		logger.Infof("Updating TaskRun, result id: %s", resultID)
	} else {
		if trResult, err = r.client.CreateResult(ctx, &pb.CreateResultRequest{
			Result: trResult,
		}); err != nil {
			logger.Errorf("Error creating TaskRun Result: %v", err)
			return err
		}
		path, err := annotationPath(trResult.GetName(), path, "add")
		if err != nil {
			logger.Errorf("Error jsonpatch for TaskRun Result %s: %v", trResult.GetName(), err)
			return err
		}
		r.pipelineclientset.TektonV1beta1().TaskRuns(tr.Namespace).Patch(tr.Name, types.JSONPatchType, path)
		logger.Infof("Creating a new result: %s", trResult.GetName())
	}

	return nil
}

// AnnotationPath creates a jsonpatch path used for adding results_id to Result
// annotations field.
func annotationPath(resultID string, path string, op string) ([]byte, error) {
	patches := []jsonpatch.JsonPatchOperation{{
		Operation: op,
		Path:      path,
		Value:     resultID,
	}}
	return json.Marshal(patches)
}
