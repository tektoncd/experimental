package pipelinerun

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
	pipelineRunLister listers.PipelineRunLister
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

	pr, err := r.pipelineclientset.TektonV1beta1().PipelineRuns(namespace).Get(name, metav1.GetOptions{})
	if errors.IsNotFound(err) {
		logger.Errorf("PipelineRun in work queue no longer exists: %v", err)
		return nil
	}
	if err != nil {
		logger.Errorf("Error retrieving PipelineRun: %v", err)
		return err
	}

	logger.Info("Recieving new PipelineRun")
	prProto, err := convert.ToPipelineRunProto(pr)
	if err != nil {
		logger.Errorf("Error converting PipelineRun to its corresponding proto: %v", err)
		return err
	}

	prResult := &pb.Result{
		Executions: []*pb.Execution{{
			Execution: &pb.Execution_PipelineRun{prProto},
		}},
	}

	if resultID, ok := prProto.GetMetadata().GetAnnotations()[idName]; ok {
		prResult.Name = resultID
		if _, err = r.client.UpdateResult(ctx, &pb.UpdateResultRequest{
			Name:   resultID,
			Result: prResult,
		}); err != nil {
			logger.Errorf("Error updating PipelineRun: %v", err)
			return err
		}
		logger.Infof("Updating PipelineRun, result id: ", resultID)
	} else {
		if prResult, err = r.client.CreateResult(ctx, &pb.CreateResultRequest{
			Result: prResult,
		}); err != nil {
			logger.Errorf("Error creating PipelineRun Result: %v", err)
			return err
		}
		path, err := annotationPath(prResult.GetName(), path, "add")
		if err != nil {
			logger.Errorf("Error jsonpatch for PipelineRun Result %s: %v", prResult.GetName(), err)
			return err
		}
		r.pipelineclientset.TektonV1beta1().PipelineRuns(pr.Namespace).Patch(pr.Name, types.JSONPatchType, path)
		logger.Infof("Creating a new result: %s", prResult.GetName())
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
