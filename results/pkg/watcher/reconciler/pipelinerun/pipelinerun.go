package pipelinerun

import (
	"context"

	"github.com/tektoncd/experimental/results/pkg/watcher/convert"
	"github.com/tektoncd/experimental/results/pkg/watcher/reconciler/common"
	pb "github.com/tektoncd/experimental/results/proto/v1alpha1/results_go_proto"
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

	if resultID, ok := prProto.GetMetadata().GetAnnotations()[common.IDName]; ok {
		result, err := r.client.GetResult(ctx, &pb.GetResultRequest{Name: resultID})
		if err != nil {
			logger.Fatalf("Error retrieving result %s: %v", resultID, err)
		}
		found := false
		for idx, execution := range result.Executions {
			remotePr := execution.GetPipelineRun()
			if remotePr != nil && remotePr.Metadata.Namespace == pr.Namespace && remotePr.Metadata.Name == pr.Name {
				found = true
				result.Executions[idx] = &pb.Execution{Execution: &pb.Execution_PipelineRun{prProto}}
			}
		}
		if !found {
			result.Executions = append(result.Executions, &pb.Execution{Execution: &pb.Execution_PipelineRun{prProto}})
		}
		if _, err := r.client.UpdateResult(ctx, &pb.UpdateResultRequest{
			Name:   resultID,
			Result: result,
		}); err != nil {
			logger.Errorf("Error updating PipelineRun: %v", err)
			return err
		}
		logger.Infof("Updating PipelineRun, result id: ", resultID)
	} else {
		prResult, err := r.client.CreateResult(ctx, &pb.CreateResultRequest{
			Result: &pb.Result{
				Executions: []*pb.Execution{{
					Execution: &pb.Execution_PipelineRun{prProto},
				}},
			},
		})
		if err != nil {
			logger.Errorf("Error creating PipelineRun Result: %v", err)
			return err
		}
		path, err := common.AnnotationPath(prResult.GetName(), common.Path, "add")
		if err != nil {
			logger.Errorf("Error jsonpatch for PipelineRun Result %s: %v", prResult.GetName(), err)
			return err
		}
		if _, err := r.pipelineclientset.TektonV1beta1().PipelineRuns(pr.Namespace).Patch(pr.Name, types.JSONPatchType, path); err != nil {
			logger.Errorf("Error apply the patch to PipelineRun: %v", err)
			return err
		}
		logger.Infof("Creating a new result: %s", prResult.GetName())
	}

	return nil
}
