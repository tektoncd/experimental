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
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	fakek8s "k8s.io/client-go/kubernetes/fake"
	"knative.dev/pkg/configmap/informer"
	logtesting "knative.dev/pkg/logging/testing"
	rtesting "knative.dev/pkg/reconciler/testing"
	"knative.dev/pkg/system"
)

func TestNewConfigStore(t *testing.T) {
	ctx, _ := rtesting.SetupFakeContext(t)

	ns := system.Namespace()
	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      TrustedTaskConfig,
			Namespace: ns,
		},
	}
	fakekubeclient := fakek8s.NewSimpleClientset(cm)
	cmw := informer.NewInformedWatcher(fakekubeclient, system.Namespace())

	store := NewConfigStore(logtesting.TestLogger(t))
	store.WatchConfigs(cmw)
	if err := cmw.Start(ctx.Done()); err != nil {
		t.Fatalf("Error starting configmap.Watcher %v", err)
	}

	// Check that with an empty configmap we get the default values.
	if diff := cmp.Diff(store.Load(), defaultConfig()); diff != "" {
		t.Errorf("unexpected data: %v", diff)
	}

	cm.Data = map[string]string{}
	cm.Data[CosignPubKey] = "newPath"

	cmw.ManualWatcher.OnChange(cm)

	expected := &Config{CosignKey: "newPath"}
	cfg := FromContext(store.ToContext(ctx))

	if diff := cmp.Diff(cfg, expected); diff != "" {
		t.Error(diff)
	}

}
