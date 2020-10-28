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
	"fmt"
	"text/template"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"sigs.k8s.io/yaml"
)

var (
	summaryTmpl = template.Must(template.New("").Funcs(template.FuncMap{
		"yaml": func(o interface{}) (string, error) {
			b, err := yaml.Marshal(o)
			if err != nil {
				return "", err
			}
			return string(b), nil
		},
	}).ParseFiles("template.md"))
)

func render(tr *v1beta1.TaskRun) (string, error) {
	b := new(bytes.Buffer)
	if err := summaryTmpl.ExecuteTemplate(b, "template.md", tr); err != nil {
		return "", fmt.Errorf("template.Execute: %w", err)
	}
	return b.String(), nil
}
