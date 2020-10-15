/*
Copyright 2020 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package main

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"flag"
	"log"
	"time"

	"github.com/tektoncd/experimental/results/pkg/convert"
	creds "github.com/tektoncd/experimental/results/pkg/grpc"
	pb "github.com/tektoncd/experimental/results/proto/proto"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	"go.uber.org/zap"
	jsonpatch "gomodules.xyz/jsonpatch/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/configmap"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/injection"
	"knative.dev/pkg/injection/sharedmain"
	"knative.dev/pkg/signals"
	_ "knative.dev/pkg/system/testing"
)

var (
	apiAddr  = flag.String("api_addr", "localhost:50051", "Address of API server to report to")
	authMode = flag.String("auth_mode", "", "Authentication mode to use when making requests. If not set, no additional credentials will be used in the request. Valid values: [google]")
)

const (
	path   = "/metadata/annotations/results.tekton.dev~1id"
	idName = "results.tekton.dev/id"
)

func main() {
	flag.Parse()

	// Setup TLS certs to the server.
	certs, err := x509.SystemCertPool()
	if err != nil {
		log.Fatalf("error loading cert pool: %v", err)
	}
	cred := credentials.NewTLS(&tls.Config{
		RootCAs: certs,
	})

	opts := []grpc.DialOption{
		grpc.WithAuthority(*apiAddr),
		grpc.WithTransportCredentials(cred),
		grpc.WithBlock(),
		grpc.WithTimeout(30 * time.Second),
	}

	// Add in additional credentials to requests if desired.
	switch *authMode {
	case "google":
		opts = append(opts, grpc.WithDefaultCallOptions(grpc.PerRPCCredentials(creds.Google())))
	}

	log.Printf("dialing %s...\n", *apiAddr)
	conn, err := grpc.Dial(*apiAddr, opts...)
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	log.Println("connected!")

	sharedmain.MainWithContext(injection.WithNamespaceScope(signals.NewContext(), ""), "watcher", func(ctx context.Context, cmw configmap.Watcher) *controller.Impl {
		client := pb.NewResultsClient(conn)
		return newController(ctx, cmw, client)
	})
}

type reconciler struct {
	logger            *zap.SugaredLogger
	client            pb.ResultsClient
	taskRunLister     listers.TaskRunLister
	pipelineclientset versioned.Interface
}

func (r *reconciler) Reconcile(ctx context.Context, key string) error {
	// 1. Get resource object from key
	// 2. Convert the object to the corresponding Result object.
	// 3. Create/Update the Result Object

	r.logger.Infof("reconciling resource key: %s", key)
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		r.logger.Errorf("invalid resource key: %s", key)
		return err
	}

	// Get the resource object with this namespace/name
	taskRun, terr := r.pipelineclientset.TektonV1beta1().TaskRuns(namespace).Get(name, metav1.GetOptions{})
	pipelineRun, perr := r.pipelineclientset.TektonV1beta1().PipelineRuns(namespace).Get(name, metav1.GetOptions{})

	if errors.IsNotFound(terr) && errors.IsNotFound(perr) {
		r.logger.Infof("Task/Pipeline run %q in work queue no longer exists", key)
		return err
	}
	if terr != nil {
		r.logger.Errorf("Error retrieving TaskRun %q: %s", name, terr)
		return terr
	}
	if perr != nil {
		r.logger.Errorf("Error retrieving PipelineRun %q: %s", name, perr)
		return perr
	}

	r.logger.Infof("Receiving new Result %s/%s", namespace, name)

	var result *pb.Result = nil

	val := ""
	ok := false
	resultType := "OPAQUE"

	if taskRun != nil {
		resultType = "TASKRUN"
		p, err := convert.ToTaskRunProto(taskRun)
		if err != nil {
			r.logger.Errorf("Error converting to proto: %v", err)
			return err
		}
		result = &pb.Result{
			Executions: []*pb.Execution{{
				Execution: &pb.Execution_TaskRun{p},
			}},
		}
		val, ok = p.GetMetadata().GetAnnotations()[idName]
	}
	if pipelineRun != nil {
		resultType = "PIPELINERUN"
		p, err := convert.ToPipelineRunProto(pipelineRun)
		if err != nil {
			r.logger.Errorf("Error converting to proto: %v", err)
			return err
		}
		result = &pb.Result{
			Executions: []*pb.Execution{{
				Execution: &pb.Execution_PipelineRun{p},
			}},
		}
		val, ok = p.GetMetadata().GetAnnotations()[idName]
	}

	if ok {
		result.Name = val
		if _, err := r.client.UpdateResult(ctx, &pb.UpdateResultRequest{
			Name:   val,
			Result: result,
		}); err != nil {
			r.logger.Errorf("Error updating Tekton Result(%s): %s\n%v", resultType, val, err)
			return err
		}
		r.logger.Infof("Sending updates for Tekton Result(%s): %s/%s (result: %s)", resultType, namespace, name, val)
	} else {
		result, err = r.client.CreateResult(ctx, &pb.CreateResultRequest{
			Result: result,
		})
		if err != nil {
			r.logger.Errorf("Error creating Tekton Result(%s): %s/%s\n%v", resultType, namespace, name, err)
			return err
		}
		path, err := annotationPath(result.GetName(), path, "add")
		if err != nil {
			r.logger.Errorf("Error jsonpatch for Tekton Result(%s): %s/%s\n%v", resultType, namespace, name, err)
			return err
		}
		r.pipelineclientset.TektonV1beta1().TaskRuns(namespace).Patch(name, types.JSONPatchType, path)
		r.logger.Infof("Creating a new Tekton Result(%s): %s/%s (result: %s)", resultType, namespace, name, result.GetName())
	}

	return err
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
