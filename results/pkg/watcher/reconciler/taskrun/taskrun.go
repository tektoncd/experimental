package taskrun

import (
	"context"

	"github.com/tektoncd/experimental/results/pkg/watcher/convert"
	"github.com/tektoncd/experimental/results/pkg/watcher/reconciler/common"
	pb "github.com/tektoncd/experimental/results/proto/proto"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/logging"
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

	if resultID, ok := trProto.GetMetadata().GetAnnotations()[common.IDName]; ok {
		result, err := r.client.GetResult(ctx, &pb.GetResultRequest{Name: resultID})
		if err != nil {
			logger.Fatalf("Error retrieving result %s: %v", resultID, err)
		}
		found := false
		for idx, execution := range result.Executions {
			remoteTr := execution.GetTaskRun()
			if remoteTr != nil && remoteTr.Metadata.Namespace == tr.Namespace && remoteTr.Metadata.Name == tr.Name {
				found = true
				result.Executions[idx] = &pb.Execution{Execution: &pb.Execution_TaskRun{trProto}}
			}
		}
		if !found {
			result.Executions = append(result.Executions, &pb.Execution{Execution: &pb.Execution_TaskRun{trProto}})
		}
		if _, err := r.client.UpdateResult(ctx, &pb.UpdateResultRequest{
			Name:   resultID,
			Result: result,
		}); err != nil {
			logger.Errorf("Error updating TaskRun: %v", err)
			return err
		}
		logger.Infof("Updating TaskRun, result id: %s", resultID)
	} else {
		trResult, err := r.client.CreateResult(ctx, &pb.CreateResultRequest{
			Result: &pb.Result{
				Executions: []*pb.Execution{{
					Execution: &pb.Execution_TaskRun{trProto},
				}},
			},
		})
		if err != nil {
			logger.Errorf("Error creating TaskRun Result: %v", err)
			return err
		}
		path, err := common.AnnotationPath(trResult.GetName(), common.Path, "add")
		if err != nil {
			logger.Errorf("Error jsonpatch for TaskRun Result %s: %v", trResult.GetName(), err)
			return err
		}
		if _, err := r.pipelineclientset.TektonV1beta1().TaskRuns(tr.Namespace).Patch(tr.Name, types.JSONPatchType, path); err != nil {
			logger.Errorf("Error apply the patch to TaskRun: %v", err)
			return err
		}
		logger.Infof("Creating a new result: %s", trResult.GetName())
	}

	return nil
}
