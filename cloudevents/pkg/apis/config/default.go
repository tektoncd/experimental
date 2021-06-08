package config

import (
	corev1 "k8s.io/api/core/v1"
	"os"
)

const (
	defaultCloudEventsSinkKey  = "default-cloud-events-sink"
	DefaultCloudEventSinkValue = ""
)

// Defaults holds the default configurations
// +k8s:deepcopy-gen=true
type Defaults struct {
	DefaultCloudEventsSink string
}

// GetDefaultsConfigName returns the name of the configmap containing all
// defined defaults.
func GetDefaultsConfigName() string {
	if e := os.Getenv("CLOUDEVENTS_CONFIG_DEFAULTS_NAME"); e != "" {
		return e
	}
	return "config-defaults"
}

// Equals returns true if two Configs are identical
func (cfg *Defaults) Equals(other *Defaults) bool {
	if cfg == nil && other == nil {
		return true
	}

	if cfg == nil || other == nil {
		return false
	}

	return other.DefaultCloudEventsSink == cfg.DefaultCloudEventsSink
}

func NewDefaultsFromConfigMap(config *corev1.ConfigMap) (*Defaults, error) {
	return NewDefaultsFromMap(config.Data)
}

// NewDefaultsFromMap returns a CEConfig given a map corresponding to a ConfigMap
func NewDefaultsFromMap(cfgMap map[string]string) (*Defaults, error) {
	tc := Defaults{
		DefaultCloudEventsSink: DefaultCloudEventSinkValue,
	}

	if defaultCloudEventsSink, ok := cfgMap[defaultCloudEventsSinkKey]; ok {
		tc.DefaultCloudEventsSink = defaultCloudEventsSink
	}

	return &tc, nil
}
