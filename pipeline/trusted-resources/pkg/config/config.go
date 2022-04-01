/*
Copyright 2021 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package config

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	cm "knative.dev/pkg/configmap"
)

// Config holds the collection of configurations that we attach to contexts.
// Configmap named with "config-trusted-resources" should include "signing-secret-path" as key in the data
type Config struct {
	// CosignKey defines the name of the key in configmap data
	CosignKey string
}

const (
	// CosignPubKey is the name of the key in configmap data
	CosignPubKey = "signing-secret-path"
	// SecretPath is the default path of cosign public key
	DefaultSecretPath = "/etc/signing-secrets/cosign.pub"
	// TrustedTaskConfig is the name of the trusted resources configmap
	TrustedTaskConfig = "config-trusted-resources"
)

func defaultConfig() *Config {
	return &Config{
		CosignKey: DefaultSecretPath,
	}
}

// NewConfigFromMap creates a Config from the supplied map
func NewConfigFromMap(data map[string]string) (*Config, error) {
	cfg := defaultConfig()
	if err := cm.Parse(data,
		cm.AsString(CosignPubKey, &cfg.CosignKey),
	); err != nil {
		return nil, fmt.Errorf("failed to parse data: %w", err)
	}
	return cfg, nil
}

// NewConfigFromConfigMap creates a Config from the supplied ConfigMap
func NewConfigFromConfigMap(configMap *corev1.ConfigMap) (*Config, error) {
	return NewConfigFromMap(configMap.Data)
}
