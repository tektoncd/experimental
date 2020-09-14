package main

import (
	"context"

	pb "github.com/tektoncd/experimental/results/proto/proto"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline"
	pipelineclient "github.com/tektoncd/pipeline/pkg/client/injection/client"
	taskruninformer "github.com/tektoncd/pipeline/pkg/client/injection/informers/pipeline/v1beta1/taskrun"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
)

// newController creates a Controller with provided context and configmap
func newController(ctx context.Context, cmw configmap.Watcher, client pb.ResultsClient) *controller.Impl {
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
