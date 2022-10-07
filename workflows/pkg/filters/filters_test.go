package filters_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/workflows/pkg/apis/workflows/v1alpha1"
	"github.com/tektoncd/experimental/workflows/pkg/filters"
	triggersv1beta1 "github.com/tektoncd/triggers/pkg/apis/triggers/v1beta1"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
)

func TestConvertFilters(t *testing.T) {
	gitRef := "gitRef"
	custom := "custom"
	tcs := []struct {
		name    string
		filters *v1alpha1.Filters
		want    []*triggersv1beta1.TriggerInterceptor
	}{{
		name: "nil filter",
	}, {
		name: "gitref filter",
		filters: &v1alpha1.Filters{
			GitRef: &v1alpha1.GitRef{Regex: "^main$"},
		},
		want: []*triggersv1beta1.TriggerInterceptor{{
			Name:   &gitRef,
			Ref:    triggersv1beta1.InterceptorRef{Name: "cel", Kind: triggersv1beta1.ClusterInterceptorKind},
			Params: []triggersv1beta1.InterceptorParams{{Name: "filter", Value: v1.JSON{Raw: []uint8(`"body.ref.split('/')[2].matches(^main$)"`)}}},
		}},
	}, {
		name: "custom filter",
		filters: &v1alpha1.Filters{
			Custom: []v1alpha1.Custom{{CEL: "body.action in ['opened', 'synchronize', 'reopened']"}},
		},
		want: []*triggersv1beta1.TriggerInterceptor{{
			Name:   &custom,
			Ref:    triggersv1beta1.InterceptorRef{Name: "cel", Kind: triggersv1beta1.ClusterInterceptorKind},
			Params: []triggersv1beta1.InterceptorParams{{Name: "filter", Value: v1.JSON{Raw: []uint8(`"body.action in ['opened', 'synchronize', 'reopened']"`)}}},
		}},
	}}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			got, err := filters.ToInterceptors(tc.filters)
			if err != nil {
				t.Errorf("unexpected error %s", err)
			}
			if d := cmp.Diff(tc.want, got); d != "" {
				t.Errorf("got wrong triggerInterceptor: %s", d)
			}
		})
	}
}
