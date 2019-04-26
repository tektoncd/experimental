package github

import (
	"context"
	"reflect"
	"testing"

	cloudevents "github.com/cloudevents/sdk-go"
	gh "github.com/google/go-github/github"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	tkn "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1alpha1"
)

func TestNewTriggerService(t *testing.T) {
	type args struct {
		ghClient     *gh.Client
		tektonClient *tkn.TektonV1alpha1Client
	}
	tests := []struct {
		name string
		args args
		want *Trigger
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewTrigger(tt.args.ghClient, tt.args.tektonClient); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewTriggerService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrigger_Handler(t *testing.T) {
	type fields struct {
		ghClient     *gh.Client
		tektonClient *tkn.TektonV1alpha1Client
	}
	type args struct {
		ctx   context.Context
		event cloudevents.Event
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Trigger{
				ghClient:     tt.fields.ghClient,
				tektonClient: tt.fields.tektonClient,
			}
			if err := tr.Handler(tt.args.ctx, tt.args.event); (err != nil) != tt.wantErr {
				t.Errorf("Trigger.Handler() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTrigger_getPipeline(t *testing.T) {
	type fields struct {
		ghClient     *gh.Client
		tektonClient *tkn.TektonV1alpha1Client
	}
	type args struct {
		ctx  context.Context
		repo *gh.PushEventRepository
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *v1alpha1.Pipeline
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Trigger{
				ghClient:     tt.fields.ghClient,
				tektonClient: tt.fields.tektonClient,
			}
			got, err := tr.getPipeline(tt.args.ctx, tt.args.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("Trigger.getPipeline() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Trigger.getPipeline() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrigger_getPipelineRun(t *testing.T) {
	type fields struct {
		ghClient     *gh.Client
		tektonClient *tkn.TektonV1alpha1Client
	}
	type args struct {
		ctx  context.Context
		repo *gh.PushEventRepository
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *v1alpha1.PipelineRun
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Trigger{
				ghClient:     tt.fields.ghClient,
				tektonClient: tt.fields.tektonClient,
			}
			got, err := tr.getPipelineRun(tt.args.ctx, tt.args.repo)
			if (err != nil) != tt.wantErr {
				t.Errorf("Trigger.getPipelineRun() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Trigger.getPipelineRun() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrigger_getGHFileContent(t *testing.T) {
	type fields struct {
		ghClient     *gh.Client
		tektonClient *tkn.TektonV1alpha1Client
	}
	type args struct {
		ctx  context.Context
		repo *gh.PushEventRepository
		path string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    *string
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Trigger{
				ghClient:     tt.fields.ghClient,
				tektonClient: tt.fields.tektonClient,
			}
			got, err := tr.getGHFileContent(tt.args.ctx, tt.args.repo, tt.args.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("Trigger.getGHFileContent() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("Trigger.getGHFileContent() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTrigger_createPipelineRun(t *testing.T) {
	type fields struct {
		ghClient     *gh.Client
		tektonClient *tkn.TektonV1alpha1Client
	}
	type args struct {
		pipeline    *v1alpha1.Pipeline
		pipelinerun *v1alpha1.PipelineRun
		pe          *gh.PushEvent
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tr := &Trigger{
				ghClient:     tt.fields.ghClient,
				tektonClient: tt.fields.tektonClient,
			}
			if err := tr.createPipelineRun(tt.args.pipeline, tt.args.pipelinerun, tt.args.pe); (err != nil) != tt.wantErr {
				t.Errorf("Trigger.createPipelineRun() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
