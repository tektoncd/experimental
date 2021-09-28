/*
Copyright 2021 The Tekton Authors

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

package pipelineruns

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/tektoncd/experimental/remote-resolution/pkg/apis/resolution/v1alpha1"
	rrclientset "github.com/tektoncd/experimental/remote-resolution/pkg/client/clientset/versioned"
	rrlisters "github.com/tektoncd/experimental/remote-resolution/pkg/client/listers/resolution/v1alpha1"
	"github.com/tektoncd/experimental/remote-resolution/pkg/resolution"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	clientset "github.com/tektoncd/pipeline/pkg/client/clientset/versioned"
	listers "github.com/tektoncd/pipeline/pkg/client/listers/pipeline/v1beta1"
	triggerslisters "github.com/tektoncd/triggers/pkg/client/listers/triggers/v1alpha1"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/cache"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
)

const resolverAnnotationKey = "resolution.tekton.dev/resolver"
const resolutionTypeLabelKey = "resolution.tekton.dev/type"

type Reconciler struct {
	// Implements reconciler.LeaderAware
	reconciler.LeaderAwareFuncs

	kubeClientSet            kubernetes.Interface
	pipelineClientSet        clientset.Interface
	pipelinerunLister        listers.PipelineRunLister
	resourceRequestLister    rrlisters.ResourceRequestLister
	resourceRequestClientSet rrclientset.Interface
	clusterInterceptorLister triggerslisters.ClusterInterceptorLister

	mode ResolutionMode
}

type ResolutionMode string

var (
	ResolutionModeRR = ResolutionMode("rr")
	ResolutionModeCI = ResolutionMode("ci")
)

func NewResolutionMode(s string) ResolutionMode {
	switch ResolutionMode(s) {
	case ResolutionModeRR:
		return ResolutionModeRR
	case ResolutionModeCI:
		return ResolutionModeCI
	default:
		return ResolutionModeRR
	}
}

var _ controller.Reconciler = &Reconciler{}

func (r *Reconciler) Reconcile(ctx context.Context, key string) error {
	prNamespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		err = &resolution.ErrorInvalidResourceKey{Key: key, Original: err}
		return controller.NewPermanentError(err)
	}

	pr, err := r.pipelinerunLister.PipelineRuns(prNamespace).Get(name)
	if err != nil {
		// the resource has been deleted or we dont have access
		// to it. either way, consider it permanent.
		err = &resolution.ErrorGettingResource{Kind: "pipelinerun", Key: key, Original: err}
		return controller.NewPermanentError(err)
	}

	if pr.IsDone() || pr.Status.PipelineSpec != nil {
		return nil
	}

	if r.mode == ResolutionModeRR {
		return r.resolveViaResourceRequest(ctx, pr)
	} else {
		return r.resolveViaClusterInterceptor(ctx, pr)
	}
}

type ResolutionInterceptorRequest struct {
	Params map[string]string `json:"params"`
}

type ResolutionInterceptorResponse struct {
	Resolved string `json:"resolved"`
	Status   Status `json:"status"`
}

type Status struct {
	Code    codes.Code `json:"code"`
	Message string     `json:"message"`
}

func (r *Reconciler) resolveViaClusterInterceptor(ctx context.Context, pr *v1beta1.PipelineRun) error {
	if len(pr.Annotations) != 0 && pr.Annotations[resolverAnnotationKey] != "" {
		resolverName := pr.Annotations[resolverAnnotationKey]
		resolverClusterInterceptorName := resolverName + "-resolver"
		if resolverName != "" {
			ci, err := r.clusterInterceptorLister.Get(resolverClusterInterceptorName)
			if err != nil {
				return r.OnError(ctx, pr, fmt.Errorf("error fetching ClusterInterceptor %q: %v", resolverClusterInterceptorName, err))
			}
			if ci == nil {
				return r.OnError(ctx, pr, fmt.Errorf("no ClusterInterceptor named %q", resolverClusterInterceptorName))
			}
			if ci.Status.Address == nil || ci.Status.Address.URL == nil {
				return r.OnError(ctx, pr, fmt.Errorf("ClusterInterceptor %q not addressable", resolverClusterInterceptorName))
			}
			urlStr := ci.Status.Address.URL.String()
			logging.FromContext(ctx).Infof("POST %q", urlStr)

			req := ResolutionInterceptorRequest{
				Params: extractResolverParams(resolverName, pr.Annotations),
			}

			reqBytes, err := json.Marshal(req)
			if err != nil {
				return r.OnError(ctx, pr, fmt.Errorf("error serializing request to resolver %q: %v", resolverName, err))
			}

			httpresp, err := http.Post(urlStr, "application/json", bytes.NewReader(reqBytes))
			if err != nil {
				return r.OnError(ctx, pr, fmt.Errorf("error posting to resolver service %q: %v", resolverName, err))
			}
			defer httpresp.Body.Close()

			resp := ResolutionInterceptorResponse{}
			dec := json.NewDecoder(httpresp.Body)
			if err := dec.Decode(&resp); err != nil && err != io.EOF {
				return r.OnError(ctx, pr, fmt.Errorf("error unmarshalling response from resolver %q: %v", resolverName, err))
			}

			if resp.Status.Code != codes.OK {
				return r.OnError(ctx, pr, fmt.Errorf("error response from resolver %q: code %d: %s", resolverName, resp.Status.Code, resp.Status.Message))
			}
			return r.patchPR(ctx, pr, resp.Resolved)
		}
	}
	return nil
}

func extractResolverParams(resolverName string, annotations map[string]string) map[string]string {
	if len(annotations) != 0 && resolverName != "" {
		paramPrefix := resolverName + "."
		params := map[string]string{}
		for key, value := range annotations {
			if strings.HasPrefix(key, paramPrefix) {
				params[strings.TrimPrefix(key, paramPrefix)] = value
			}
		}
		return params
	}
	return nil
}

func (r *Reconciler) resolveViaResourceRequest(ctx context.Context, pr *v1beta1.PipelineRun) error {
	name := pr.Name
	prNamespace := pr.Namespace
	// TODO(sbwsg): this naming scheme could run over k8s metadata
	// name limits and is generally not very robust. Choose
	// something better - either totally randomized or a way to
	// inject more information into the short name length.
	rrName := "pr-" + name
	rr, _ := r.resourceRequestLister.ResourceRequests(prNamespace).Get(rrName)

	if rr == nil {
		return r.CreateResourceRequest(ctx, rrName, pr)
	}

	if rr.Status.GetCondition(apis.ConditionSucceeded).IsUnknown() {
		return nil
	}

	if rr.Status.GetCondition(apis.ConditionSucceeded).IsTrue() {
		r.patchPR(ctx, pr, rr.Status.Data)
	} else {
		message := rr.Status.GetCondition(apis.ConditionSucceeded).GetMessage()
		err := resolution.NewError(resolution.ReasonResolutionFailed, errors.New(message))
		return r.OnError(ctx, pr, err)
	}

	return nil
}

func (r *Reconciler) patchPR(ctx context.Context, pr *v1beta1.PipelineRun, data string) error {
	pipeline := &v1beta1.Pipeline{}
	err := json.Unmarshal([]byte(data), pipeline)
	if err == nil {
		resolution.CopyPipelineMetaToPipelineRun(&pipeline.ObjectMeta, pr)
		pr.Status.PipelineSpec = &pipeline.Spec
	} else {
		return r.OnError(ctx, pr, fmt.Errorf("not a valid pipeline: %v", err))
	}
	if _, err := PatchResolvedPipelineRun(ctx, r.kubeClientSet, r.pipelineClientSet, pr); err != nil {
		logging.FromContext(ctx).Errorf("error patching resolved pipelinerun: %v", err)
		// We don't mark the pipelinerun failed here because error
		// responses from the api server might be transient.
		return err
	}
	return nil
}

func (r *Reconciler) CreateResourceRequest(ctx context.Context, rrName string, pr *v1beta1.PipelineRun) error {
	if len(pr.Annotations) != 0 && pr.Annotations[resolverAnnotationKey] != "" {
		resolverName := pr.Annotations[resolverAnnotationKey]
		params := extractResolverParams(resolverName, pr.Annotations)

		rr := &v1alpha1.ResourceRequest{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "resolution.tekton.dev/v1alpha1",
				Kind:       "ResourceRequest",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      rrName,
				Namespace: pr.Namespace,
				Labels: map[string]string{
					resolutionTypeLabelKey: resolverName,
				},
			},
			Spec: v1alpha1.ResourceRequestSpec{
				Parameters: params,
			},
		}
		appendOwnerReference(rr, pr)
		_, err := r.resourceRequestClientSet.ResolutionV1alpha1().ResourceRequests(rr.Namespace).Create(ctx, rr, metav1.CreateOptions{})
		if err != nil {
			return r.OnError(ctx, pr, err)
		}
	}
	return nil
}

func (r *Reconciler) OnError(ctx context.Context, pr *v1beta1.PipelineRun, err error) error {
	if err != nil {
		updateErr := UpdatePipelineRunWithError(ctx, r.pipelineClientSet, pr.Namespace, pr.Name, err)
		if updateErr != nil {
			return err
		}
		return controller.NewPermanentError(err)
	}
	return nil
}

func ResolvePipelineRun(ctx context.Context, kClient kubernetes.Interface, pClient clientset.Interface, pr *v1beta1.PipelineRun) (*v1beta1.PipelineRun, error) {
	req := resolution.PipelineRunResolutionRequest{
		KubeClientSet:     kClient,
		PipelineClientSet: pClient,
		PipelineRun:       pr,
	}

	if err := req.Resolve(ctx); err != nil {
		return nil, err
	}

	resolution.CopyPipelineMetaToPipelineRun(req.ResolvedPipelineMeta, pr)
	pr.Status.PipelineSpec = req.ResolvedPipelineSpec

	return pr, nil
}

// UpdatePipelineRunWithError updates a PipelineRun with a resolution error.
// Available publicly so that unit tests can leverage resolution machinery
// without relying on Patch, which our current fakes don't support.
//
// Returns an error if updating the PipelineRun doesn't work.
func UpdatePipelineRunWithError(ctx context.Context, client clientset.Interface, namespace, name string, resolutionError error) error {
	key := fmt.Sprintf("%s/%s", namespace, name)
	reason, err := PipelineRunResolutionReasonError(resolutionError)
	latestGenerationPR, err := client.TektonV1beta1().PipelineRuns(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		logging.FromContext(ctx).Warnf("error getting pipelinerun %q to update it as failed: %v", key, err)
		return err
	}

	latestGenerationPR.Status.MarkFailed(reason, resolutionError.Error())
	_, err = client.TektonV1beta1().PipelineRuns(namespace).UpdateStatus(ctx, latestGenerationPR, metav1.UpdateOptions{})
	if err != nil {
		logging.FromContext(ctx).Warnf("error marking pipelinerun %q as failed: %v", key, err)
		return err
	}
	return nil
}

// PipelineRunResolutionReasonError extracts the reason and underlying error
// embedded in the given resolution.Error, or returns some sane defaults
// if the error isn't a resolution.Error.
func PipelineRunResolutionReasonError(err error) (string, error) {
	reason := resolution.ReasonPipelineRunResolutionFailed
	resolutionError := err

	if e, ok := err.(*resolution.Error); ok {
		reason = e.Reason
		resolutionError = e.Unwrap()
	}

	return reason, resolutionError
}

func appendOwnerReference(rr *v1alpha1.ResourceRequest, pr *v1beta1.PipelineRun) {
	apiVersion := pr.TypeMeta.APIVersion
	kind := pr.TypeMeta.Kind

	if apiVersion == "" {
		apiVersion = "tekton.dev/v1beta1"
	}

	if kind == "" {
		kind = "PipelineRun"
	}

	rr.ObjectMeta.OwnerReferences = append(rr.ObjectMeta.OwnerReferences, metav1.OwnerReference{
		APIVersion: apiVersion,
		Kind:       kind,
		Name:       pr.ObjectMeta.Name,
		UID:        pr.ObjectMeta.UID,
		Controller: truePointer(),
	})
}

func truePointer() *bool {
	val := true
	return &val
}
