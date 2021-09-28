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

package noopref

import (
	"context"
)

const pipeline = `
{
    "apiVersion": "tekton.dev/v1beta1",
    "kind": "Pipeline",
    "metadata": {
        "name": "p"
    },
    "spec": {
        "tasks": [
            {
                "name": "fetch-from-git",
                "taskSpec": {
                    "metadata": {},
                    "spec": null,
                    "steps": [
                        {
                            "image": "alpine@sha256:69704ef328d05a9f806b6b8502915e6a0a4faa4d72018dc42343f511490daf8a",
                            "name": "",
                            "resources": {}
                        }
                    ]
                }
            }
        ]
    }
}
`

type Resolver struct{}

func (r *Resolver) Initialize(ctx context.Context) error {
	return nil
}

func (r *Resolver) GetName() string {
	return "No-Op"
}

func (r *Resolver) GetSelector() map[string]string {
	return map[string]string{
		"resolution.tekton.dev/type": "noop",
	}
}

func (r *Resolver) ValidateParams(params map[string]string) error {
	return nil
}

func (r *Resolver) Resolve(params map[string]string) (string, map[string]string, error) {
	return pipeline, nil, nil
}
