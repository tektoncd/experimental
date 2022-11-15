package filters

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"knative.dev/pkg/ptr"
)

func ToInterceptors(f *v1alpha1.Filters) ([]*triggersv1beta1.TriggerInterceptor, error) {
	if f == nil {
		return nil, nil
	}
	var out []*triggersv1beta1.TriggerInterceptor
	if f.GitRef != nil {
		i, err := gitRefToInterceptor(*f.GitRef)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	for _, c := range f.Custom {
		i, err := customToInterceptor(c)
		if err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, nil
}

// gitRefToInterceptor returns an interceptor that filters events to those affecting
// the specified gitRef.
func gitRefToInterceptor(gr v1alpha1.GitRef) (*triggersv1beta1.TriggerInterceptor, error) {
	_, err := regexp.Compile(gr.Regex)
	if err != nil {
		return nil, fmt.Errorf("invalid regular expression: %w", err)
	}

	// For right now we're assuming that the event body
	// contains a top-level "refs" field of the form "refs/heads/main"
	celFilter := fmt.Sprintf("body.ref.split('/')[2].matches('%s')", gr.Regex)
	celFilterToJSON, err := ToV1JSON(celFilter)
	if err != nil {
		return nil, err
	}
	gitRefInterceptor := triggersv1beta1.TriggerInterceptor{
		Name: ptr.String("gitRef"),
		Ref: triggersv1beta1.InterceptorRef{
			Name: "cel",
			Kind: "ClusterInterceptor",
		},
		Params: []triggersv1beta1.InterceptorParams{{
			Name:  "filter",
			Value: celFilterToJSON,
		}},
	}
	return &gitRefInterceptor, nil
}

// customToInterceptor returns an interceptor with the custom filtering logic
func customToInterceptor(c v1alpha1.Custom) (*triggersv1beta1.TriggerInterceptor, error) {
	celFilterToJSON, err := ToV1JSON(c.CEL)
	if err != nil {
		return nil, err
	}
	return &triggersv1beta1.TriggerInterceptor{
		Name: ptr.String("custom"),
		Ref: triggersv1beta1.InterceptorRef{
			Name: "cel",
			Kind: "ClusterInterceptor",
		},
		Params: []triggersv1beta1.InterceptorParams{{
			Name:  "filter",
			Value: celFilterToJSON,
		}},
	}, nil
}

// ToV1JSON is a wrapper around json.Marshal to easily convert to the Kubernetes apiextensionsv1.JSON type
func ToV1JSON(v interface{}) (v1.JSON, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return v1.JSON{}, err
	}
	return v1.JSON{
		Raw: b,
	}, nil
}
