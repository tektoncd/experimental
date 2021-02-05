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

package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"text/template"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned/scheme"
)

// Take a file path, parse it as YAML -> v1beta1.Task

func readTask(name string) (v1beta1.Task, error) {

	var task v1beta1.Task
	dat, err := ioutil.ReadFile(name)
	if err != nil {
		return task, fmt.Errorf("error reading file: %w", err)
	}
	if _, _, err := scheme.Codecs.UniversalDeserializer().Decode(dat, nil, &task); err != nil {
		return task, fmt.Errorf("error decoding task: %w", err)
	}

	return task, nil
}

// Use Go templates to print v1beta1.Task as Markdown

func printTask(w io.Writer, task v1beta1.Task) error {
	if task.Spec.Params != nil {

		tmpl := template.Must(template.New("test").Parse(`# {{.Name}}
## Install the Task
kubectl apply -f https://raw.githubusercontent.com/tektoncd/catalog/master/task/{{.Name}}/0.1/{{.Name}}.yaml
### Parameters:-
`))
		if err := tmpl.Execute(w, task); err != nil {
			return fmt.Errorf("error executing the template: %w", err)
		}
	}

	if task.Spec.Params != nil {

		t := `{{range .}}- {{.Name}}, {{.Description}}, (default: {{.Default}}) 
{{end}}`
		tmpl := template.Must(template.New("test").Parse(t))
		if err := tmpl.Execute(w, task.Spec.Params); err != nil {
			return fmt.Errorf("error executing the template: %w", err)
		}
	}

	if task.Spec.Resources.Inputs != nil {

		t := `### Resources:-
{{range .}}- {{.ResourceDeclaration.Name}}, {{.ResourceDeclaration.Type}}
{{end}}`
		tmpl := template.Must(template.New("test").Parse(t))
		if err := tmpl.Execute(w, task.Spec.Resources.Inputs); err != nil {
			return fmt.Errorf("error executing the template: %w", err)
		}
	}

	if task.Spec.Resources.Outputs != nil {

		t := `### Output:-
{{range .}}- {{.ResourceDeclaration.Name}}, {{.ResourceDeclaration.Type}}
{{end}}`
		tmpl := template.Must(template.New("test").Parse(t))
		if err := tmpl.Execute(w, task.Spec.Resources.Outputs); err != nil {
			return fmt.Errorf("error executing the template: %w", err)
		}
	}
	return nil
}

func main() {

	flag.Parse()

	task, err := readTask(os.Args[1])
	if err != nil {
		log.Fatalln("failed to read Task:", err)
	}

	if err := printTask(os.Stdout, task); err != nil {
		log.Fatalln("failed to render Task:", err)
	}
}

// TODO: tests!
