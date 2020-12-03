// Package cel provides definitions for defining the Results CEL environment.
package cel

import (
	"github.com/google/cel-go/cel"
	"github.com/google/cel-go/checker/decls"
	"github.com/google/cel-go/common/types"
	"github.com/google/cel-go/common/types/ref"
	ppb "github.com/tektoncd/experimental/results/proto/pipeline/v1beta1/pipeline_go_proto"
	pb "github.com/tektoncd/experimental/results/proto/v1alpha2/results_go_proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// NewEnv returns the CEL environment for Results, loading in definitions for
// known types.
func NewEnv() (*cel.Env, error) {
	return cel.NewEnv(
		cel.Types(&pb.Result{}, &ppb.PipelineRun{}, &ppb.TaskRun{}),
		cel.Declarations(decls.NewVar("result", decls.NewObjectType("tekton.results.v1alpha2.Result"))),
		cel.Declarations(decls.NewVar("taskrun", decls.NewObjectType("tekton.pipeline.v1beta1.TaskRun"))),
		cel.Declarations(decls.NewVar("pipelinerun", decls.NewObjectType("tekton.pipeline.v1beta1.PipelineRun"))),
	)
}

// ParseFilter creates a CEL program based on the given filter string.
func ParseFilter(env *cel.Env, filter string) (cel.Program, error) {
	if filter == "" {
		return allowAll{}, nil
	}

	ast, issues := env.Compile(filter)
	if issues != nil && issues.Err() != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error parsing filter: %v", issues.Err())
	}

	prg, err := env.Program(ast)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "error creating filter query evaluator: %v", err)
	}
	return prg, nil
}

// allowAll is a CEL program implementation that always returns true.
type allowAll struct{}

func (allowAll) Eval(interface{}) (ref.Val, *cel.EvalDetails, error) {
	return types.Bool(true), nil, nil
}
