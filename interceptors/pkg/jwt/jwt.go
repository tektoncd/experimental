package jwt

import (
	"context"

	"github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"github.com/tektoncd/triggers/pkg/interceptors"
	"google.golang.org/grpc/codes"

	triggersv1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1alpha1"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"

	"github.com/lestrrat-go/jwx/jwk"
	"github.com/lestrrat-go/jwx/jwt"
)

const (
	HeaderParam = "header"
)

type Interceptor struct {
	KubeClientSet kubernetes.Interface
	Logger        *zap.SugaredLogger
}

type JWTInterceptor struct {
	Claims   map[string]interface{} `json:"claims,omitempty"`
	JWKSUrl  string                 `json:"jwks_url,omitempty"`
	Issuer   string                 `json:"issuer,omitempty"`
	Audience string                 `json:"aud,omitempty"`
}

// NewInterceptor creates a prepopulated Interceptor.
func NewInterceptor(k kubernetes.Interface, l *zap.SugaredLogger) v1alpha1.InterceptorInterface {
	return &Interceptor{
		Logger:        l,
		KubeClientSet: k,
	}
}

func (w *Interceptor) Process(ctx context.Context, r *triggersv1.InterceptorRequest) *triggersv1.InterceptorResponse {
	p := JWTInterceptor{}
	if err := interceptors.UnmarshalParams(r.InterceptorParams, &p); err != nil {
		return interceptors.Failf(codes.InvalidArgument, "failed to parse interceptor params: %w", err)
	}

	if p.JWKSUrl == "" {
		return interceptors.Failf(codes.InvalidArgument, "invalid jwks_url argument provided")
	}

	headers := interceptors.Canonical(r.Header)
	authHeader := headers.Get("Authorization")

	if authHeader == "" {
		return interceptors.Failf(codes.InvalidArgument, "expected jwt token in Authorization header")
	}

	parseOptions := []jwt.ParseOption{
		// validate when parsing the token
		jwt.WithValidate(true),
	}

	if p.JWKSUrl != "" {
		keyset, err := jwk.Fetch(ctx, p.JWKSUrl)
		if err != nil {
			return interceptors.Failf(codes.FailedPrecondition, "failed to fetch JWKS keyset: %w", err)
		}
		parseOptions = append(parseOptions, jwt.WithKeySet(keyset))
	}
	if p.Issuer != "" {
		parseOptions = append(parseOptions, jwt.WithIssuer(p.Issuer))
	}
	if p.Audience != "" {
		jwt.WithAudience(p.Audience)
	}

	if p.Claims != nil {
		for k, v := range p.Claims {
			parseOptions = append(parseOptions, jwt.WithClaimValue(k, v))
		}
	}

	_, err := jwt.Parse(
		[]byte(authHeader[7:]),
		parseOptions...,
	)

	if err != nil {
		return interceptors.Failf(codes.FailedPrecondition, "failed to parse JWT: %w", err)
	}

	return &v1alpha1.InterceptorResponse{
		Continue: true,
	}
}
