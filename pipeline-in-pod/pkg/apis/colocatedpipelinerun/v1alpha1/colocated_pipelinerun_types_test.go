package v1alpha1_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	cprv1alpha1 "github.com/tektoncd/experimental/pipeline-in-pod/pkg/apis/colocatedpipelinerun/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestGetTimeout(t *testing.T) {
	cpr := cprv1alpha1.ColocatedPipelineRun{
		Spec: cprv1alpha1.ColocatedPipelineRunSpec{
			Timeouts: &v1beta1.TimeoutFields{Pipeline: &metav1.Duration{Duration: time.Duration(15 * time.Second)}},
		},
	}
	timeout := cpr.PipelineTimeout(context.Background())
	expected := time.Duration(15 * time.Second)
	if d := cmp.Diff(expected, timeout); d != "" {
		t.Errorf("wrong timeout: %s", d)
	}
}
