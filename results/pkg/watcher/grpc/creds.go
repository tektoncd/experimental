package grpc

import (
	"context"
	"fmt"

	"google.golang.org/api/idtoken"
	"google.golang.org/grpc/credentials"
)

// Google provides an authenticated transport for use against Google APIs
// (e.g. Cloud Run, Identity Aware Proxy, API Gateway, etc.) using the
// incoming gRPC server URI as the token audience.
// See https://pkg.go.dev/google.golang.org/api/idtoken for more details.
func Google(opts ...idtoken.ClientOption) credentials.PerRPCCredentials {
	return &googleCreds{
		opts: opts,
	}
}

type googleCreds struct {
	opts []idtoken.ClientOption
}

// GetRequestMetadata gets the current request metadata, refreshing
// tokens if required. This should be called by the transport layer on
// each request, and the data should be populated in headers or other
// context. If a status code is returned, it will be used as the status
// for the RPC. uri is the URI of the entry point for the request.
// When supported by the underlying implementation, ctx can be used for
// timeout and cancellation. Additionally, RequestInfo data will be
// available via ctx to this call.
func (c *googleCreds) GetRequestMetadata(ctx context.Context, uri ...string) (map[string]string, error) {
	out := map[string]string{}
	for _, u := range uri {
		tokenSource, err := idtoken.NewTokenSource(ctx, u, c.opts...)
		if err != nil {
			return nil, fmt.Errorf("idtoken.NewTokenSource(%s): %v", u, err)
		}
		token, err := tokenSource.Token()
		if err != nil {
			return nil, fmt.Errorf("TokenSource.Token(%s): %v", u, err)
		}
		out["authorization"] = "Bearer " + token.AccessToken
	}
	return out, nil
}

// RequireTransportSecurity indicates whether the credentials requires
// transport security.
func (googleCreds) RequireTransportSecurity() bool {
	return true
}
