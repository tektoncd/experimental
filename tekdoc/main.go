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
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tektoncd/pipeline/pkg/client/clientset/versioned/scheme"
	"io"
	"io/ioutil"
	"log"
	"os"
	"text/template"
)

var (
	filename = flag.String("f", "", "Name of the file to parse")
)

// TODO: take a file path, parse it as YAML -> v1alpha1.Task

func read(name string) (v1alpha1.Task, error) {

	var task v1alpha1.Task
	dat, err := ioutil.ReadFile(name)
	if err != nil {
		return task, fmt.Errorf("error reading file: %w", err)
	}
	if _, _, err := scheme.Codecs.UniversalDeserializer().Decode(dat, nil, &task); err != nil {
		return task, fmt.Errorf("error decoding task: %w", err)
	}

	return task, nil
}

// TODO: use Go templates to print v1alpha1.Task as Markdown

func printTask(w io.Writer, task v1alpha1.Task) error{
	if task.Spec.Inputs != nil {

		tmpl := template.Must(template.New("test").Parse(`# {{.Name}}
## Install the Task
kubectl apply -f https://raw.githubusercontent.com/tektoncd/catalog/master/{{.Name}}/{{.Name}}.yaml
### Input:-
`))
		if err := tmpl.Execute(w, task); err != nil {
			return fmt.Errorf("error executing the template: %v", err)
		}
	}

	if task.Spec.Inputs.Params != nil {

		t := `{{range .}}- {{.Name}}, {{.Description}}
{{end}}`
		tmpl := template.Must(template.New("test").Parse(t))
		if err := tmpl.Execute(w, task.Spec.Inputs.Params); err != nil {
			return fmt.Errorf("error executing the template: %v", err)
		}
	}

	if task.Spec.Inputs.Resources != nil {

		t := `### Resources:-
{{range .}}- {{.ResourceDeclaration.Name}}, {{.ResourceDeclaration.Type}}
{{end}}`
		tmpl := template.Must(template.New("test").Parse(t))
		if err := tmpl.Execute(w, task.Spec.Inputs.Resources); err != nil {
			return fmt.Errorf("error executing the template: %v", err)
		}
	}

	if task.Spec.Outputs != nil {

		t := `### Output:-
{{range .}}- {{.ResourceDeclaration.Name}}, {{.ResourceDeclaration.Type}}
{{end}}`
		tmpl := template.Must(template.New("test").Parse(t))
		err := tmpl.Execute(w, task.Spec.Outputs.Resources)
		if err != nil {
			return fmt.Errorf("error executing the template: %v", err)
		}
	}
	return nil
}

func main() {

	flag.Parse()

	task, err := read(*filename)
	if err != nil {
		log.Fatalln(err)
	}

	err = printTask(os.Stdout, task)
	if err != nil {
		log.Fatalln(err)
	}
}

// TODO: tests!
