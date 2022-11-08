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

// Code generated by injection-gen. DO NOT EDIT.

package fake

import (
	context "context"

	fake "github.com/tektoncd/experimental/workflows/pkg/client/injection/informers/factory/fake"
	gitrepository "github.com/tektoncd/experimental/workflows/pkg/client/injection/informers/workflows/v1alpha1/gitrepository"
	controller "knative.dev/pkg/controller"
	injection "knative.dev/pkg/injection"
)

var Get = gitrepository.Get

func init() {
	injection.Fake.RegisterInformer(withInformer)
}

func withInformer(ctx context.Context) (context.Context, controller.Informer) {
	f := fake.Get(ctx)
	inf := f.Workflows().V1alpha1().GitRepositories()
	return context.WithValue(ctx, gitrepository.Key{}, inf), inf.Informer()
}
