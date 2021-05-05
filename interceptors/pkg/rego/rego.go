package rego

import (
	"context"
	"encoding/json"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"google.golang.org/grpc/codes"

	"github.com/open-policy-agent/opa/ast"
	"github.com/open-policy-agent/opa/rego"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/interceptors"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
)

const (
	HeaderParam = "header"
)

type Interceptor struct {
	KubeClientSet kubernetes.Interface
	Logger        *zap.SugaredLogger
}

type RegoInterceptor struct {
	Module string `json:"module,omitempty"`
	Query  string `json:"query,omitempty"`
}

// NewInterceptor creates a prepopulated Interceptor.
func NewInterceptor(k kubernetes.Interface, l *zap.SugaredLogger) v1alpha1.InterceptorInterface {
	return &Interceptor{
		Logger:        l,
		KubeClientSet: k,
	}
}

func (w *Interceptor) Process(ctx context.Context, r *triggersv1.InterceptorRequest) *triggersv1.InterceptorResponse {
	p := RegoInterceptor{}
	if err := interceptors.UnmarshalParams(r.InterceptorParams, &p); err != nil {
		return interceptors.Failf(codes.InvalidArgument, "failed to parse interceptor params: %w", err)
	}
	var body map[string]interface{}

	if err := json.Unmarshal([]byte(r.Body), &body); err != nil {
		return interceptors.Failf(codes.FailedPrecondition, "unable to marshal interceptor body to json: %w", err)
	}

	inputBody := map[string]interface{}{
		"body":       body,
		"header":     r.Header,
		"extensions": r.Extensions,
	}
	compiler, err := ast.CompileModules(map[string]string{
		"tekton.rego": p.Module,
	})
	if err != nil {
		return interceptors.Failf(codes.FailedPrecondition, "failed to compile rego module: %w", err)
	}
	regoFilter := rego.New(
		rego.Query(p.Query),
		rego.Compiler(compiler),
		rego.Input(inputBody))

	rs, err := regoFilter.Eval(ctx)
	if err != nil {
		return &triggersv1.InterceptorResponse{
			Continue: false,
			Status: triggersv1.Status{
				Message: err.Error(),
				Code:    codes.Aborted,
			},
		}
	}

	// query result
	return &triggersv1.InterceptorResponse{
		Continue: rs[0].Expressions[0].Value.(bool),
	}
}
