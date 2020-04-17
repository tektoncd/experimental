package main

import (
	"context"
	"flag"
	"log"

	pb "github.com/tektoncd/experimental/results/proto/proto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	taskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1alpha1/taskrun"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1alpha1"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/signals"
)

var (
	apiAddr   = flag.String("api_addr", "localhost:50051", "Address of API server to report to")
	namespace = flag.String("namespace", corev1.NamespaceAll, "Namespace to restrict informer to. Optional, defaults to all namespaces.")
)

func main() {
	flag.Parse()

	// Set up a connection to the server.
	conn, err := grpc.Dial(*apiAddr, grpc.WithInsecure(), grpc.WithBlock())
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	client := pb.NewResultsClient(conn)

	sharedmain.MainWithContext(injection.WithNamespaceScope(signals.NewContext(), *namespace), "watcher", func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		logger := logging.FromContext(ctx)
		taskRunInformer := taskruninformer.Get(ctx)

		c := &reconciler{
			logger:        logger,
			taskRunLister: taskRunInformer.Lister(),
			client:        client,
		}
		impl := controller.NewImpl(c, c.logger, pipeline.PipelineRunControllerName)

		taskRunInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
			AddFunc:    impl.EnqueueControllerOf,
			UpdateFunc: controller.PassNew(impl.EnqueueControllerOf),
		})

		return impl
	})
}

type reconciler struct {
	logger        *zap.SugaredLogger
	client        pb.ResultsClient
	taskRunLister listers.TaskRunLister
}

func (r *reconciler) Reconcile(ctx context.Context, key string) error {
	r.logger.Infof("reconciling resource key: %s", key)

	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		r.logger.Errorf("invalid resource key: %s", key)
		return nil
	}

	// Get the Task Run resource with this namespace/name
	tr, err := r.taskRunLister.TaskRuns(namespace).Get(name)
	if errors.IsNotFound(err) {
		// The resource no longer exists, in which case we stop processing.
		r.logger.Infof("task run %q in work queue no longer exists", key)
		return nil
	} else if err != nil {
		r.logger.Errorf("Error retrieving TaskRun %q: %s", name, err)
		return err
	}

	r.logger.Infof("Sending update for %s/%s (uid %s)", namespace, name, tr.UID)

	// Send the new status of the TaskRun to the API server.
	if _, err := r.client.UpdateTaskRun(ctx, &pb.UpdateTaskRunRequest{
		TaskRun: &pb.TaskRun{
			Uid:       string(tr.UID),
			Name:      tr.Name,
			Namespace: tr.Namespace,
			// TODO: More robust/scalable translation.
		},
	}); err != nil {
		r.logger.Error("Error updating TaskRun %s: %v", name, err)
		return err
	}

	return nil
}
