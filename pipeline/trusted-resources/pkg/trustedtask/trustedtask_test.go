package trustedtask

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	faketekton "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/fake"
	"github.com/tektoncd/pipeline/test/diff"
	"go.uber.org/zap/zaptest"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	"knative.dev/pkg/logging"
)

func init() {
	os.Setenv("SYSTEM_NAMESPACE", nameSpace)
	os.Setenv("WEBHOOK_SERVICEACCOUNT_NAME", serviceAccount)
}

func Test_verifyTask(t *testing.T) {
	ctx := logging.WithLogger(context.Background(), zaptest.NewLogger(t).Sugar())
	k8sClient := fakek8s.NewSimpleClientset(sa)
	tektonClient := faketekton.NewSimpleClientset()
	signer, secretpath, _ := getSignerFromFile(t, ctx, k8sClient)
	ctx = setupContext(ctx, k8sClient, secretpath)

	task := v1beta1.Task{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "tekton.dev/v1beta1",
			Kind:       "Task"},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
			Namespace: nameSpace,
		},
		Spec: *taskSpecTest,
	}

	expectedSignature, err := SignInterface(signer, task)
	if err != nil {
		t.Errorf("SignInterface() err: %v", err)
	}

	tests := []struct {
		name        string
		task        v1beta1.Task
		expectedErr bool
	}{{
		name: "non-tampered Task",
		task: v1beta1.Task{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "tekton.dev/v1beta1",
				Kind:       "Task"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: nameSpace,
			},
			Spec: *taskSpecTest,
		},
		expectedErr: false,
	}, {
		name: "tampered Task - different name",
		task: v1beta1.Task{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "tekton.dev/v1beta1",
				Kind:       "Task"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "no-foo",
				Namespace: nameSpace,
			},
			Spec: *taskSpecTest,
		},
		expectedErr: true,
	}, {
		name: "tampered Task - unexpected labels",
		task: v1beta1.Task{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "tekton.dev/v1beta1",
				Kind:       "Task"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: nameSpace,
				Labels: map[string]string{
					"unexpected-label": "foo-label",
				},
			},
			Spec: *taskSpecTest,
		},
		expectedErr: true,
	}, {
		name: "tampered Task - missing namespace",
		task: v1beta1.Task{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "tekton.dev/v1beta1",
				Kind:       "Task"},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: nameSpace,
				Labels: map[string]string{
					"unexpected-label": "foo-label",
				},
			},
			Spec: *taskSpecTest,
		},
		expectedErr: true,
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			signedTask := test.task.DeepCopy()
			if signedTask.Annotations == nil {
				signedTask.Annotations = map[string]string{}
			}
			signedTask.Annotations[SignatureAnnotation] = expectedSignature
			trustedTask := &TrustedTask{Task: *signedTask}
			if err := trustedTask.verifyTask(ctx, k8sClient, tektonClient); (err != nil) != test.expectedErr {
				t.Errorf("verifyTask() got err: %v", err)
			}
		})
	}
}

func Test_getSignature(t *testing.T) {
	tests := []struct {
		name        string
		metadata    metav1.ObjectMeta
		expectedErr error
	}{{
		name: "valid signature",
		metadata: metav1.ObjectMeta{
			Annotations: map[string]string{
				SignatureAnnotation: "MEUCIQDjaR9SBdqMwMZVkxZqLdFCPsAq4sFbZqeRJDS/8JsiXAIgDzph3QjQzBP6V+EXW75RemETSjzPb4P75XTqdlnDOo0=",
			},
		},
		expectedErr: nil,
	}, {
		name: "missing signature",
		metadata: metav1.ObjectMeta{
			Annotations: map[string]string{},
		},
		expectedErr: fmt.Errorf("signature is missing"),
	}, {
		name: "illegal signature to decode",
		metadata: metav1.ObjectMeta{
			Annotations: map[string]string{
				SignatureAnnotation: "abcde",
			},
		},
		expectedErr: fmt.Errorf("illegal base64 data at input byte 4"),
	}}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := getSignature(test.metadata); err != test.expectedErr {
				if d := cmp.Diff(test.expectedErr.Error(), err.Error()); d != "" {
					t.Error(diff.PrintWantGot(d))
				}
			}
		})
	}
}
