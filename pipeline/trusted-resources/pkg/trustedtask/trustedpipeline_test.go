package trustedtask

import (
	"context"
	"os"
	"testing"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	faketekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	"go.uber.org/zap/zaptest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	"knative.dev/pkg/logging"
)

func init() {
	os.Setenv("SYSTEM_NAMESPACE", nameSpace)
	os.Setenv("WEBHOOK_SERVICEACCOUNT_NAME", serviceAccount)
}

func Test_verifyPipeline(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), zaptest.NewLogger(t).Sugar())
	k8sClient := fakek8s.NewSimpleClientset(sa)
	tektonClient := faketekton.NewSimpleClientset()
	signer, secretpath := getSignerFromFile(t, ctx, k8sClient)
	ctx = setupContext(ctx, k8sClient, secretpath)

	pipeline := v1beta1.Pipeline{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Pipeline"},
		ObjectMeta: metav1.ObjectMeta{
			Name:        "foo",
			Namespace:   nameSpace,
			Annotations: map[string]string{},
		},
		Spec: v1beta1.PipelineSpec{
			Tasks: []v1beta1.PipelineTask{{
				Name: "foo",
			}},
		},
	}

	expectedSignature, err := SignInterface(signer, pipeline)
	if err != nil {
		t.Errorf("SignInterface() err: %v", err)
	}

	tests := []struct {
		name        string
		pipeline    v1beta1.Pipeline
		expectedErr bool
	}{{
		name:        "non-tampered Pipeline",
		pipeline:    pipeline,
		expectedErr: false,
	}, {
		name: "tampered Pipeline - different name",
		pipeline: v1beta1.Pipeline{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "tekton.dev/v1beta1",
				Kind:       "Pipeline"},
			ObjectMeta: metav1.ObjectMeta{
				Name:        "no-foo",
				Namespace:   nameSpace,
				Annotations: map[string]string{},
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{{
					Name: "foo",
				}},
			},
		},
		expectedErr: true,
	}, {
		name: "tampered Pipeline - unexpected annotations",
		pipeline: v1beta1.Pipeline{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "tekton.dev/v1beta1",
				Kind:       "Pipeline"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: nameSpace,
				Annotations: map[string]string{
					"unexpected": "foo",
				},
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{{
					Name: "foo",
				}},
			},
		},
		expectedErr: true,
	}, {
		name: "tampered Pipeline - missing namespace",
		pipeline: v1beta1.Pipeline{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "tekton.dev/v1beta1",
				Kind:       "Pipeline"},
			ObjectMeta: metav1.ObjectMeta{
				Name: "foo",
				Annotations: map[string]string{
					"unexpected": "foo",
				},
			},
			Spec: v1beta1.PipelineSpec{
				Tasks: []v1beta1.PipelineTask{{
					Name: "foo",
				}},
			},
		},
		expectedErr: true,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			signedPipeline := test.pipeline.DeepCopy()
			if signedPipeline.Annotations == nil {
				signedPipeline.Annotations = map[string]string{}
			}
			signedPipeline.Annotations[SignatureAnnotation] = expectedSignature
			trustedPipeline := &TrustedPipeline{Pipeline: *signedPipeline}
			if err := trustedPipeline.verifyPipeline(ctx, k8sClient, tektonClient); (err != nil) != test.expectedErr {
				t.Errorf("verifyPipeline() got err: %v", err)
			}
		})
	}
}
