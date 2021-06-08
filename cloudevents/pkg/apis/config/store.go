package config

import (
	"context"
	"knative.dev/pkg/configmap"
)

type cfgKey struct{}

// CEConfig holds the collection of configurations that we attach to contexts.
// +k8s:deepcopy-gen=false
type CEConfig struct {
	Defaults *Defaults
}

// FromContext extracts a CEConfig from the provided context.
func FromContext(ctx context.Context) *CEConfig {
	x, ok := ctx.Value(cfgKey{}).(*CEConfig)
	if ok {
		return x
	}
	return nil
}

// FromContextOrDefaults is like FromContext, but when no CEConfig is attached it
// returns a CEConfig populated with the defaults for each of the CEConfig fields.
func FromContextOrDefaults(ctx context.Context) *CEConfig {
	if cfg := FromContext(ctx); cfg != nil {
		return cfg
	}
	defaults, _ := NewDefaultsFromMap(map[string]string{})
	return &CEConfig{
		Defaults: defaults,
	}
}

// ToContext attaches the provided CEConfig to the provided context, returning the
// new context with the CEConfig attached.
func ToContext(ctx context.Context, c *CEConfig) context.Context {
	return context.WithValue(ctx, cfgKey{}, c)
}

// CCStore is a typed wrapper around configmap.Untyped store to handle our configmaps.
// +k8s:deepcopy-gen=false
type CCStore struct {
	*configmap.UntypedStore
}

// NewStore creates a new store of Configs and optionally calls functions when ConfigMaps are updated.
func NewStore(logger configmap.Logger, onAfterStore ...func(name string, value interface{})) *CCStore {
	store := &CCStore{
		UntypedStore: configmap.NewUntypedStore(
			"defaults",
			logger,
			configmap.Constructors{
				GetDefaultsConfigName(): NewDefaultsFromConfigMap,
			},
			onAfterStore...,
		),
	}

	return store
}

// ToContext attaches the current CEConfig state to the provided context.
func (s *CCStore) ToContext(ctx context.Context) context.Context {
	return ToContext(ctx, s.Load())
}

// Load creates a CEConfig from the current config state of the CCStore.
func (s *CCStore) Load() *CEConfig {
	defaults := s.UntypedLoad(GetDefaultsConfigName())
	if defaults == nil {
		defaults, _ = NewDefaultsFromMap(map[string]string{})
	}

	return &CEConfig{
		Defaults: defaults.(*Defaults).DeepCopy(),
	}
}
