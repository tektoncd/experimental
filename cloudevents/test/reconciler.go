/*
Copyright 2022 The Tekton Authors

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

package test

import (
	"context"
	"testing"
	"time"

	"github.com/tektoncd/experimental/cloudevents/pkg/apis/config"
	pipelinetest "github.com/tektoncd/pipeline/test"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/clock"
	"k8s.io/client-go/tools/record"
	"knative.dev/pkg/configmap"
	cminformer "knative.dev/pkg/configmap/informer"
	"knative.dev/pkg/controller"
	"knative.dev/pkg/logging"
	"knative.dev/pkg/reconciler"
	"knative.dev/pkg/system"
	_ "knative.dev/pkg/system/testing"
)

var (
	now       = time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)
	testClock = clock.NewFakePassiveClock(now)
)

type controllerBuilder func(clock.PassiveClock) func(context.Context, configmap.Watcher) *controller.Impl

func ensureConfigurationConfigMapsExist(d *pipelinetest.Data) {
	var defaultsExists bool
	for _, cm := range d.ConfigMaps {
		if cm.Name == config.GetDefaultsConfigName() {
			defaultsExists = true
		}
	}
	if !defaultsExists {
		d.ConfigMaps = append(d.ConfigMaps, &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{Name: config.GetDefaultsConfigName(), Namespace: system.Namespace()},
			Data:       map[string]string{},
		})
	}
}

func getResourceController(t *testing.T, d pipelinetest.Data, builder controllerBuilder) (pipelinetest.Assets, func()) {
	// unregisterMetrics()
	ctx, _ := SetupFakeContext(t)
	ctx, cancel := context.WithCancel(ctx)
	ensureConfigurationConfigMapsExist(&d)
	c, informers := pipelinetest.SeedTestData(t, ctx, d)
	configMapWatcher := cminformer.NewInformedWatcher(c.Kube, system.Namespace())

	ctl := builder(testClock)(ctx, configMapWatcher)

	if la, ok := ctl.Reconciler.(reconciler.LeaderAware); ok {
		la.Promote(reconciler.UniversalBucket(), func(reconciler.Bucket, types.NamespacedName) {})
	}
	if err := configMapWatcher.Start(ctx.Done()); err != nil {
		t.Fatalf("error starting configmap watcher: %v", err)
	}

	return pipelinetest.Assets{
		Logger:     logging.FromContext(ctx),
		Controller: ctl,
		Clients:    c,
		Informers:  informers,
		Recorder:   controller.GetEventRecorder(ctx).(*record.FakeRecorder),
		Ctx:        ctx,
	}, cancel
}

type ReconcileTest struct {
	pipelinetest.Data `json:"inline"`
	Test              *testing.T
	TestAssets        pipelinetest.Assets
	Cancel            func()
}

func (rt ReconcileTest) ReconcileRun(t *testing.T, namespace string, resourceName string) pipelinetest.Clients {
	rt.Test.Helper()
	c := rt.TestAssets.Controller
	clients := rt.TestAssets.Clients

	reconcileError := c.Reconciler.Reconcile(rt.TestAssets.Ctx, namespace+"/"+resourceName)
	if reconcileError != nil {
		rt.Test.Fatalf("Error reconciling: %s", reconcileError)
	}

	for _, a := range clients.Kube.Actions() {
		aVerb := a.GetVerb()
		if aVerb != "get" && aVerb != "list" && aVerb != "watch" {
			t.Errorf("Expected only read actions to be logged in the kubeclient, got %s", aVerb)
		}
	}

	return clients
}

func NewReconcileTest(data pipelinetest.Data, builder controllerBuilder, t *testing.T) *ReconcileTest {
	t.Helper()
	testAssets, cancel := getResourceController(t, data, builder)
	return &ReconcileTest{
		Data:       data,
		Test:       t,
		TestAssets: testAssets,
		Cancel:     cancel,
	}
}
