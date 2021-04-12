package hmac

import (
	"context"
	"fmt"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"google.golang.org/grpc/codes"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	gh "github.com/google/go-github/v31/github"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
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

type HmacInterceptor struct {
	Header    string              `json:"header,omitempty"`
	Algorithm string              `json:"algoritm,omitempty"`
	SecretRef *v1alpha1.SecretRef `json:"secretRef,omitempty"`
}

// NewInterceptor creates a prepopulated Interceptor.
func NewInterceptor(k kubernetes.Interface, l *zap.SugaredLogger) v1alpha1.InterceptorInterface {
	return &Interceptor{
		Logger:        l,
		KubeClientSet: k,
	}
}

func (w *Interceptor) Process(ctx context.Context, r *triggersv1.InterceptorRequest) *triggersv1.InterceptorResponse {
	p := HmacInterceptor{}
	if err := interceptors.UnmarshalParams(r.InterceptorParams, &p); err != nil {
		return interceptors.Failf(codes.InvalidArgument, "failed to parse interceptor params: %v", err)
	}

	if p.Header == "" {
		return interceptors.Failf(codes.InvalidArgument, "invalid header argument provided")
	}
	headers := interceptors.Canonical(r.Header)
	header := headers.Get(p.Header)
	if header == "" {
		return interceptors.Fail(codes.FailedPrecondition, "no X-Hub-Signature header set")
	}

	if p.Algorithm != "" {
		// prepend the algorithm to the header value
		header = fmt.Sprintf("%s=%s", p.Algorithm, header)
	}

	secretData, err := w.getSecret(ctx, p, r.Context.TriggerID)
	if err != nil {
		return interceptors.Fail(codes.FailedPrecondition, err.Error())
	}

	if err := gh.ValidateSignature(header, []byte(r.Body), secretData); err != nil {
		return interceptors.Fail(codes.FailedPrecondition, err.Error())
	}

	return &triggersv1.InterceptorResponse{
		Continue: true,
	}
}

func (w *Interceptor) getSecret(ctx context.Context, p HmacInterceptor, triggerId string) ([]byte, error) {
	if p.SecretRef == nil {
		return nil, fmt.Errorf("hmac interceptor secretRef is not set")
	}
	// Check the secret to see if it is empty
	if p.SecretRef.SecretKey == "" {
		return nil, fmt.Errorf("hmac interceptor secretRef.secretKey is empty")
	}

	ns, _ := triggersv1.ParseTriggerID(triggerId)
	secret, err := w.KubeClientSet.CoreV1().Secrets(ns).Get(ctx, p.SecretRef.SecretName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("error getting secret: %v", err)
	}
	return secret.Data[p.SecretRef.SecretKey], nil
}
