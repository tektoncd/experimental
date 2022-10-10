package config_test

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/tektoncd/experimental/concurrency/pkg/apis/config"
	test "github.com/tektoncd/pipeline/pkg/reconciler/testing"
	"k8s.io/apimachinery/pkg/util/sets"
	logtesting "knative.dev/pkg/logging/testing"
)

func TestStoreLoadWithContext(t *testing.T) {
	cfgMap := test.ConfigMapFromTestFile(t, "concurrency-config-all-namespaces")

	expectedCfg, err := config.NewConfigFromConfigMap(cfgMap)
	if err != nil {
		t.Fatalf("error loading configmap: %s", err)
	}

	store := config.NewStore(logtesting.TestLogger(t))
	store.OnConfigChanged(cfgMap)

	cfg := config.FromContext(store.ToContext(context.Background()))

	if d := cmp.Diff(cfg, expectedCfg); d != "" {
		t.Errorf("Unexpected config %s", d)
	}
}

func TestNewConfigFromConfigMap(t *testing.T) {
	tcs := []struct {
		filename       string
		expectedConfig *config.Config
	}{{
		filename:       "concurrency-config-all-namespaces",
		expectedConfig: &config.Config{AllowedNamespaces: sets.NewString()},
	}, {
		filename:       "concurrency-config-one-namespace",
		expectedConfig: &config.Config{AllowedNamespaces: sets.NewString("default")},
	}, {
		filename:       "concurrency-config-multiple-namespaces",
		expectedConfig: &config.Config{AllowedNamespaces: sets.NewString("default", "another-namespace")},
	}}
	for _, tc := range tcs {
		t.Run(tc.filename, func(t *testing.T) {
			cm := test.ConfigMapFromTestFile(t, tc.filename)
			cfg, err := config.NewConfigFromConfigMap(cm)
			if err != nil {
				t.Fatalf("error loading configmap: %s", err)
			}
			if d := cmp.Diff(tc.expectedConfig, cfg); d != "" {
				t.Errorf("wrong config: %s", d)
			}
		})
	}
}
