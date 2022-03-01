package config

import (
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
)

const (
	defaultCloudEventsSinkKey    = "default-cloud-events-sink"
	DefaultCloudEventSinkValue   = ""
	EventFormatLegacy            = "legacy"
	EventFormatCDEvents          = "cdevents"
	defaultCloudEventsFormatKey  = "default-cloud-events-format"
	DefaultCloudEventFormatValue = EventFormatCDEvents
)

// Defaults holds the default configurations
// +k8s:deepcopy-gen=true
type Defaults struct {
	DefaultCloudEventsSink   string
	DefaultCloudEventsFormat string
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

	return other.DefaultCloudEventsSink == cfg.DefaultCloudEventsSink &&
		other.DefaultCloudEventsFormat == cfg.DefaultCloudEventsFormat
}

func NewDefaultsFromConfigMap(config *corev1.ConfigMap) (*Defaults, error) {
	return NewDefaultsFromMap(config.Data)
}

// NewDefaultsFromMap returns a CEConfig given a map corresponding to a ConfigMap
func NewDefaultsFromMap(cfgMap map[string]string) (*Defaults, error) {
	tc := Defaults{
		DefaultCloudEventsSink:   DefaultCloudEventSinkValue,
		DefaultCloudEventsFormat: DefaultCloudEventFormatValue,
	}

	if defaultCloudEventsSink, ok := cfgMap[defaultCloudEventsSinkKey]; ok {
		tc.DefaultCloudEventsSink = defaultCloudEventsSink
	}

	if defaultCloudEventsFormat, ok := cfgMap[defaultCloudEventsFormatKey]; ok {
		switch defaultCloudEventsFormat {
		case EventFormatLegacy, EventFormatCDEvents:
			tc.DefaultCloudEventsFormat = defaultCloudEventsFormat
		default:
			return nil, fmt.Errorf("invalid value for CloudEvents format: %q", defaultCloudEventsFormat)
		}
	}

	return &tc, nil
}
