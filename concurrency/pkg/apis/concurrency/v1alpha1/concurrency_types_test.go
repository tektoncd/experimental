package v1alpha1_test

import (
	"context"
	"testing"

	"github.com/tektoncd/experimental/concurrency/pkg/apis/concurrency/v1alpha1"
)

func TestValidateConcurrencyControl(t *testing.T) {
	tcs := []struct {
		name    string
		cc      *v1alpha1.ConcurrencyControl
		wantErr bool
	}{{
		name: "valid cancel",
		cc: &v1alpha1.ConcurrencyControl{
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "Cancel",
			},
		},
	}, {
		name: "valid cancel: Lowercase",
		cc: &v1alpha1.ConcurrencyControl{
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "cancel",
			},
		},
	}, {
		name: "invalid strategy",
		cc: &v1alpha1.ConcurrencyControl{
			Spec: v1alpha1.ConcurrencySpec{
				Strategy: "not-real-strategy",
			},
		},
		wantErr: true,
	}}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.cc.Validate(context.Background())
			if (err != nil) != tc.wantErr {
				t.Errorf("wantErr was %t but got error %s", tc.wantErr, err)
			}
		})
	}
}
