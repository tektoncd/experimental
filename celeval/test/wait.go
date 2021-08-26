/*
Copyright 2019 The Tekton Authors

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
	"fmt"
	tektontest "github.com/tektoncd/pipeline/test"
	"time"

	"go.opencensus.io/trace"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
)

const (
	interval = 1 * time.Second
	timeout  = 10 * time.Minute
)

func pollImmediateWithContext(ctx context.Context, fn func() (bool, error)) error {
	return wait.PollImmediate(interval, timeout, func() (bool, error) {
		select {
		case <-ctx.Done():
			return true, ctx.Err()
		default:
		}
		return fn()
	})
}

// WaitForRunState polls the status of the Run called name from client every
// interval until inState returns `true` indicating it is done, returns an
// error or timeout. desc will be used to name the metric that is emitted to
// track how long it took for name to get into the state checked by inState.
func WaitForRunState(ctx context.Context, c *clients, name string, polltimeout time.Duration, inState tektontest.ConditionAccessorFn, desc string) error {
	metricName := fmt.Sprintf("WaitForRunState/%s/%s", name, desc)
	_, span := trace.StartSpan(context.Background(), metricName)
	defer span.End()

	ctx, cancel := context.WithTimeout(ctx, polltimeout)
	defer cancel()
	return pollImmediateWithContext(ctx, func() (bool, error) {
		r, err := c.RunClient.Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			return true, err
		}
		return inState(&r.Status)
	})
}
