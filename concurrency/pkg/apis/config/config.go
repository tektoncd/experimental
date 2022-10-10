package config

import (
	"context"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/sets"
	"knative.dev/pkg/configmap"
)

type cfgKey struct{}

const ConcurrencyConfigMapName = "concurrency-config"
const ConcurrencyNamespace = "tekton-concurrency"

// Config holds the collection of configurations that we attach to contexts.
// +k8s:deepcopy-gen=false
type Config struct {
	AllowedNamespaces sets.String
}

// Store is a typed wrapper around configmap.Untyped store to handle our configmaps.
// +k8s:deepcopy-gen=false
type Store struct {
	*configmap.UntypedStore
}

// ToContext attaches the current Config state to the provided context.
func (s *Store) ToContext(ctx context.Context) context.Context {
	return ToContext(ctx, s.Load())
}

// ToContext attaches the provided Config to the provided context, returning the
// new context with the Config attached.
func ToContext(ctx context.Context, c *Config) context.Context {
	return context.WithValue(ctx, cfgKey{}, c)
}

// FromContext extracts a Config from the provided context.
func FromContext(ctx context.Context) *Config {
	x, ok := ctx.Value(cfgKey{}).(*Config)
	if ok {
		return x
	}
	return nil
}

// NewConfigFromConfigMap parses a Config struct from a ConfigMap
func NewConfigFromConfigMap(config *corev1.ConfigMap) (*Config, error) {
	c := Config{}
	if config == nil {
		return &c, nil
	}
	if v, ok := config.Data["allowed-namespaces"]; ok {
		if v == "" {
			c.AllowedNamespaces = sets.NewString()
		} else {
			c.AllowedNamespaces = sets.NewString(strings.Split(v, ",")...)
		}
	}
	return &c, nil
}

// NewStore creates a new store of Configs and optionally calls functions when ConfigMaps are updated.
func NewStore(logger configmap.Logger) *Store {
	store := &Store{
		UntypedStore: configmap.NewUntypedStore(
			"concurrency-config",
			logger,
			configmap.Constructors{
				// Tells knative how to parse the Config struct from the concurrency configMap
				// Function signature must be func(*k8s.io/api/core/v1.ConfigMap) (... , error)
				ConcurrencyConfigMapName: NewConfigFromConfigMap,
			},
		),
	}
	return store
}

// Load creates a Config from the current config state of the Store.
func (s *Store) Load() *Config {
	c := s.UntypedLoad(ConcurrencyConfigMapName)
	return c.(*Config)
}
