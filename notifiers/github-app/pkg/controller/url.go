// Copyright 2020 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package controller

import (
	"bytes"
	"text/template"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

const defaultDashboardURL = "https://dashboard.dogfooding.tekton.dev/#/namespaces/{{ .Namespace }}/taskruns/{{ .Name }}"

func dashboardURL(tr *v1beta1.TaskRun) (string, error) {
	url, ok := tr.Annotations[key("url")]

	if !ok || len(url) == 0 {
		url = defaultDashboardURL
	}

	t, err := template.New("dashboardURL").Parse(url)

	if err != nil {
		return "", err
	}

	var tpl bytes.Buffer

	if err := t.Execute(&tpl, tr); err != nil {
		return "", err
	}

	return tpl.String(), nil
}
