package v1alpha1

import "context"

func (rr *ResourceRequest) SetDefaults(ctx context.Context) {
	if rr.TypeMeta.Kind == "" {
		rr.TypeMeta.Kind = "ResourceRequest"
	}
	if rr.TypeMeta.APIVersion == "" {
		rr.TypeMeta.APIVersion = "resolution.tekton.dev/v1alpha1"
	}
}
