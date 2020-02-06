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

package main // import "github.com/tektoncd/experimental/octant-plugin"

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/pkg/errors"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tmc/dot"
	"github.com/vmware-tanzu/octant/pkg/plugin"
	"github.com/vmware-tanzu/octant/pkg/plugin/service"
	"github.com/vmware-tanzu/octant/pkg/store"
	"github.com/vmware-tanzu/octant/pkg/view/component"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/pkg/apis/duck"
)

var (
	taskRunGVK     = v1alpha1.SchemeGroupVersion.WithKind("TaskRun")
	taskGVK        = v1alpha1.SchemeGroupVersion.WithKind("Task")
	pipelineRunGVK = v1alpha1.SchemeGroupVersion.WithKind("PipelineRun")
	pipelineGVK    = v1alpha1.SchemeGroupVersion.WithKind("Pipeline")
)

func main() {
	log.SetPrefix("")

	// Use the plugin service helper to register this plugin.
	p, err := service.Register("tektoncd", "Manage Tekton resources",
		&plugin.Capabilities{
			SupportsPrinterStatus: []schema.GroupVersionKind{taskRunGVK, taskGVK, pipelineRunGVK, pipelineGVK},
			SupportsPrinterConfig: []schema.GroupVersionKind{taskRunGVK, taskGVK, pipelineRunGVK, pipelineGVK},
			SupportsPrinterItems:  []schema.GroupVersionKind{taskRunGVK, taskGVK, pipelineRunGVK, pipelineGVK},
			SupportsTab:           []schema.GroupVersionKind{pipelineGVK},
			ActionNames:           []string{"taskrun", "pipelinerun"},
		},
		service.WithPrinter(handlePrint),
		service.WithTabPrinter(handleTabPrint),
		service.WithActionHandler(func(request *service.ActionRequest) error {
			switch request.Payload["action"] {
			case "taskrun":
				tn := request.Payload["task"].(string)
				return fmt.Errorf("got request to run task %q: %+v", tn, request.Payload)
			case "pipelinerun":
				pn := request.Payload["pipeline"].(string)
				return fmt.Errorf("got request to run pipeline %q: %+v", pn, request.Payload)
			default:
				return fmt.Errorf("unknown action %q", request.Payload["action"])
			}
		}),
	)
	if err != nil {
		log.Fatal(err)
	}
	p.Serve()
}

// handlePrint is called when Octant wants to print information about an
// object.
func handlePrint(request *service.PrintRequest) (plugin.PrintResponse, error) {
	key, err := store.KeyFromObject(request.Object)
	if err != nil {
		return plugin.PrintResponse{}, err
	}
	u, found, err := request.DashboardClient.Get(request.Context(), key)
	if err != nil {
		return plugin.PrintResponse{}, err
	}
	if !found {
		return plugin.PrintResponse{}, errors.New("not found")
	}

	switch request.Object.GetObjectKind().GroupVersionKind() {
	case taskRunGVK:
		var tr v1alpha1.TaskRun
		if err := duck.FromUnstructured(u, &tr); err != nil {
			return plugin.PrintResponse{}, nil
		}
		return printTaskRun(request.Context(), &tr, request.DashboardClient)

	case pipelineRunGVK:
		var pr v1alpha1.PipelineRun
		if err := duck.FromUnstructured(u, &pr); err != nil {
			return plugin.PrintResponse{}, nil
		}
		return printPipelineRun(&pr), nil

	case taskGVK:
		var t v1alpha1.Task
		if err := duck.FromUnstructured(u, &t); err != nil {
			return plugin.PrintResponse{}, nil
		}
		return printTask(request.Context(), &t, request.DashboardClient)

	case pipelineGVK:
		var p v1alpha1.Pipeline
		if err := duck.FromUnstructured(u, &p); err != nil {
			return plugin.PrintResponse{}, nil
		}
		return printPipeline(request.Context(), &p, request.DashboardClient)

	default:
		return plugin.PrintResponse{}, errors.Errorf("unsupported object %q", request.Object.GetObjectKind().GroupVersionKind())
	}
}

// resources returns a list of PipelineResources of the given type.
//
// It's used to populate dropdowns to select input resources when running Tasks
// or Pipelines.
func resources(ctx context.Context, client service.Dashboard, t v1alpha1.PipelineResourceType) ([]v1alpha1.PipelineResource, error) {
	ul, err := client.List(ctx, store.Key{
		APIVersion: "tekton.dev/v1alpha1",
		Kind:       "PipelineResource",
	})
	if err != nil {
		return nil, err
	}

	var prs []v1alpha1.PipelineResource
	for _, u := range ul.Items {
		var pr v1alpha1.PipelineResource
		if err := duck.FromUnstructured(&u, &pr); err != nil {
			return nil, err
		}
		if pr.Spec.Type == t {
			prs = append(prs, pr)
		}
	}
	return prs, nil
}

func printTaskRun(ctx context.Context, tr *v1alpha1.TaskRun, client service.Dashboard) (plugin.PrintResponse, error) {
	resp := plugin.PrintResponse{}
	if tr.Status.PodName != "" {
		// TODO: this 404s for timed-out or cancelled TaskRuns, since we delete the Pod...
		s := tr.Status.PodName
		ref := "../../../../workloads/pods/" + tr.Status.PodName
		resp.Status = append(resp.Status, component.SummarySection{Header: "Pod", Content: component.NewLink("Pod Name", s, ref)})
	}
	if tr.Spec.TaskRef != nil {
		s := tr.Spec.TaskRef.Name
		ref := "../../../tasks.tekton.dev/v1alpha1/" + tr.Spec.TaskRef.Name
		// TODO: handle ClusterTask
		resp.Status = append(resp.Status, component.SummarySection{Header: "Task", Content: component.NewLink("Task Name", s, ref)})

	}

	saName := "default"
	if tr.Spec.ServiceAccountName != "" {
		saName = tr.Spec.ServiceAccountName
	}
	saref := "../../../../config-and-storage/service-accounts/" + saName
	resp.Status = append(resp.Status, component.SummarySection{Header: "Service Account", Content: component.NewLink("Service Account Name", saName, saref)})

	if !tr.Status.StartTime.Time.IsZero() {
		d := tr.Status.StartTime.Time.Sub(tr.CreationTimestamp.Time)
		resp.Status = append(resp.Status, component.SummarySection{Header: "Queued", Content: component.NewText(d.String())})
	}
	if tr.IsDone() {
		d := tr.Status.CompletionTime.Time.Sub(tr.Status.StartTime.Time)
		resp.Status = append(resp.Status, component.SummarySection{Header: "Duration", Content: component.NewText(d.String())})
	}
	if tr.Spec.Timeout != nil {
		d := tr.Spec.Timeout.Duration
		resp.Status = append(resp.Status, component.SummarySection{Header: "Timeout", Content: component.NewText(d.String())})
	}

	if tr.Status.PodName != "" {
		// TODO: handle logs/links for Pods for previous attempts.
		logsCard := component.NewCard(component.TitleFromString("Pod Logs"))

		up, _, err := client.Get(ctx, store.Key{
			APIVersion: "v1",
			Kind:       "Pod",
			Name:       tr.Status.PodName,
			Namespace:  tr.Namespace,
		})
		if err != nil {
			return plugin.PrintResponse{}, err
		}
		var p corev1.Pod
		if err := duck.FromUnstructured(up, &p); err != nil {
			return plugin.PrintResponse{}, err
		}
		var containerNames []string
		for _, p := range p.Spec.Containers {
			containerNames = append(containerNames, p.Name)
		}

		logsCard.SetBody(component.NewLogs(tr.Namespace, tr.Status.PodName, containerNames))
		resp.Items = append(resp.Items, component.FlexLayoutItem{
			Width: component.WidthFull,
			View:  logsCard,
		})
	}

	return resp, nil
}

func printPipelineRun(pr *v1alpha1.PipelineRun) plugin.PrintResponse {
	resp := plugin.PrintResponse{}

	ref := "../../../pipelines.tekton.dev/v1alpha1/" + pr.Spec.PipelineRef.Name
	resp.Status = append(resp.Status, component.SummarySection{Header: "Pipeline", Content: component.NewLink("Pipeline Name", pr.Spec.PipelineRef.Name, ref)})

	saName := "default"
	if pr.Spec.ServiceAccountName != "" {
		saName = pr.Spec.ServiceAccountName
	}
	saref := "../../../../config-and-storage/service-accounts/" + saName
	resp.Status = append(resp.Status, component.SummarySection{Header: "Service Account", Content: component.NewLink("Service Account Name", saName, saref)})

	if !pr.Status.StartTime.Time.IsZero() {
		d := pr.Status.StartTime.Time.Sub(pr.CreationTimestamp.Time)
		resp.Status = append(resp.Status, component.SummarySection{Header: "Queued", Content: component.NewText(d.String())})
	}
	if pr.IsDone() {
		d := pr.Status.CompletionTime.Time.Sub(pr.Status.StartTime.Time)
		resp.Status = append(resp.Status, component.SummarySection{Header: "Duration", Content: component.NewText(d.String())})
	}
	if pr.Spec.Timeout != nil {
		d := pr.Spec.Timeout.Duration
		resp.Status = append(resp.Status, component.SummarySection{Header: "Timeout", Content: component.NewText(d.String())})
	}

	// TODO: print taskRuns and their statuses, links to taskrun page with full logs.
	return resp
}

func printTask(ctx context.Context, t *v1alpha1.Task, client service.Dashboard) (plugin.PrintResponse, error) {
	resp := plugin.PrintResponse{}

	runCard := component.NewCard(component.TitleFromString("Run This Task"))
	runCard.SetBody(component.NewText("Specify inputs to run this Task"))
	a := component.Action{
		Name:  "Run Task",
		Title: "Run Task",
		Form: component.Form{
			Fields: []component.FormField{
				component.NewFormFieldHidden("action", "taskrun"),
				component.NewFormFieldHidden("task", t.Name),
			},
		},
	}

	var iomd string
	if t.Spec.Inputs != nil {
		if len(t.Spec.Inputs.Resources) != 0 {
			iomd += "Input Resources\n\n"
			for _, r := range t.Spec.Inputs.Resources {
				iomd += fmt.Sprintf("* `%s` (%s)\n", r.Name, r.Type)

				prs, err := resources(ctx, client, r.Type)
				if err != nil {
					return plugin.PrintResponse{}, nil
				}
				cs := []component.InputChoice{{Label: "<select one>"}}
				for _, pr := range prs {
					cs = append(cs, component.InputChoice{
						Label: pr.Name,
						Value: pr.Name,
					})
				}
				a.Form.Fields = append(a.Form.Fields, component.NewFormFieldSelect(r.Name, "resource."+r.Name, cs, false))
			}
			iomd += "\n"
		}
		if len(t.Spec.Inputs.Params) != 0 {
			iomd += "Input Parameters\n\n"
			for _, r := range t.Spec.Inputs.Params {
				iomd += fmt.Sprintf("* `%s` (%s)", r.Name, r.Type)
				if r.Default != nil {
					switch r.Default.Type {
					case v1alpha1.ParamTypeString:
						iomd += fmt.Sprintf(" (default: _%s_)", r.Default.StringVal)
						a.Form.Fields = append(a.Form.Fields, component.NewFormFieldText(r.Name, "param."+r.Name, r.Default.StringVal)) // TODO: support array-type params
					case v1alpha1.ParamTypeArray:
						b, _ := json.Marshal(r.Default.ArrayVal)
						iomd += fmt.Sprintf(" (default: _%s_)", string(b))
					}
				} else if r.Type == v1alpha1.ParamTypeString {
					a.Form.Fields = append(a.Form.Fields, component.NewFormFieldText(r.Name, "param."+r.Name, "")) // TODO: support array-type params
				}
				if r.Description != "" {
					iomd += ": " + r.Description
				}
				iomd += "\n"
			}
			iomd += "\n"
		}
	}
	if t.Spec.Outputs != nil {
		iomd += "Output Resources\n\n"
		for _, r := range t.Spec.Outputs.Resources {
			iomd += fmt.Sprintf("* `%s` (%s)\n", r.Name, r.Type)
		}
		iomd += "\n"
	}

	if iomd != "" {
		ioCard := component.NewCard(component.TitleFromString("Inputs and Outputs"))
		ioCard.SetBody(component.NewMarkdownText(iomd))
		resp.Items = append(resp.Items, component.FlexLayoutItem{
			Width: component.WidthFull,
			View:  ioCard,
		})
	}
	runCard.AddAction(a)
	resp.Items = append(resp.Items, component.FlexLayoutItem{
		Width: component.WidthFull,
		View:  runCard,
	})

	// TODO: display or link to list of most recent runs.
	return resp, nil
}

func printPipeline(ctx context.Context, p *v1alpha1.Pipeline, client service.Dashboard) (plugin.PrintResponse, error) {
	resp := plugin.PrintResponse{}

	runCard := component.NewCard(component.TitleFromString("Run This Pipeline"))
	runCard.SetBody(component.NewText("Specify inputs to run this Pipeline"))
	a := component.Action{
		Name:  "Run Pipeline",
		Title: "Run Pipeline",
		Form: component.Form{
			Fields: []component.FormField{
				component.NewFormFieldHidden("action", "pipelinerun"),
				component.NewFormFieldHidden("pipeline", p.Name),
			},
		},
	}

	var iomd string
	if len(p.Spec.Params) != 0 {
		iomd += "Input Parameters\n\n"
		for _, r := range p.Spec.Params {
			iomd += fmt.Sprintf("* `%s` (%s)", r.Name, r.Type)
			if r.Default != nil {
				switch r.Default.Type {
				case v1alpha1.ParamTypeString:
					iomd += fmt.Sprintf(" (default: _%s_)", r.Default.StringVal)
					a.Form.Fields = append(a.Form.Fields, component.NewFormFieldText(r.Name, "param."+r.Name, r.Default.StringVal)) // TODO: support array-type params
				case v1alpha1.ParamTypeArray:
					b, _ := json.Marshal(r.Default.ArrayVal)
					iomd += fmt.Sprintf(" (default: _%s_)", string(b))
				}
			} else if r.Type == v1alpha1.ParamTypeString {
				a.Form.Fields = append(a.Form.Fields, component.NewFormFieldText(r.Name, "param."+r.Name, "")) // TODO: support array-type params
			}
			if r.Description != "" {
				iomd += ": " + r.Description
			}
			iomd += "\n"
		}
	}
	if len(p.Spec.Resources) != 0 {
		iomd += "Input Resources\n\n"
		for _, r := range p.Spec.Resources {
			iomd += fmt.Sprintf("* `%s` (%s)\n", r.Name, r.Type)

			prs, err := resources(ctx, client, r.Type)
			if err != nil {
				return plugin.PrintResponse{}, nil
			}
			cs := []component.InputChoice{{Label: "<select one>"}}
			for _, pr := range prs {
				cs = append(cs, component.InputChoice{
					Label: pr.Name,
					Value: pr.Name,
				})
			}
			a.Form.Fields = append(a.Form.Fields, component.NewFormFieldSelect(r.Name, "resource."+r.Name, cs, false))
		}
	}
	if len(p.Spec.Tasks) != 0 {
		var md string
		for _, t := range p.Spec.Tasks {
			ref := "/#/overview/namespace/default/custom-resources/tasks.tekton.dev/v1alpha1/" + t.TaskRef.Name // TODO: handle ClusterTask
			md += fmt.Sprintf("* [`%s`](%s)\n", t.Name, ref)
		}
		card := component.NewCard(component.TitleFromString("Tasks"))
		card.SetBody(component.NewMarkdownText(md))
		resp.Items = append(resp.Items, component.FlexLayoutItem{
			Width: component.WidthFull,
			View:  card,
		})
	}

	if iomd != "" {
		ioCard := component.NewCard(component.TitleFromString("Inputs and Outputs"))
		ioCard.SetBody(component.NewMarkdownText(iomd))
		resp.Items = append(resp.Items, component.FlexLayoutItem{
			Width: component.WidthFull,
			View:  ioCard,
		})
	}
	runCard.AddAction(a)
	resp.Items = append(resp.Items, component.FlexLayoutItem{
		Width: component.WidthFull,
		View:  runCard,
	})
	// TODO: display or link to list of most recent runs.
	return resp, nil
}

// handleTabPrint is called when Octant wants to print a tab of content.
func handleTabPrint(request *service.PrintRequest) (plugin.TabResponse, error) {
	key, err := store.KeyFromObject(request.Object)
	if err != nil {
		return plugin.TabResponse{}, err
	}
	u, found, err := request.DashboardClient.Get(request.Context(), key)
	if err != nil {
		return plugin.TabResponse{}, err
	}
	if !found {
		return plugin.TabResponse{}, errors.New("not found")
	}

	switch request.Object.GetObjectKind().GroupVersionKind() {
	case pipelineGVK:
		var p v1alpha1.Pipeline
		if err := duck.FromUnstructured(u, &p); err != nil {
			return plugin.TabResponse{}, nil
		}
		gv, err := pipelineToGraphviz(&p)
		if err != nil {
			return plugin.TabResponse{}, err
		}
		fl := component.NewFlexLayout("Visualization")
		fl.AddSections([]component.FlexLayoutItem{{
			Width: component.WidthFull,
			View:  component.NewGraphviz(gv),
		}})
		return plugin.TabResponse{
			Tab: component.NewTabWithContents(*fl),
		}, nil
	default:
		return plugin.TabResponse{}, errors.Errorf("unsupported object %q", request.Object.GetObjectKind().GroupVersionKind())
	}

}

func pipelineToGraphviz(p *v1alpha1.Pipeline) (string, error) {
	g := dot.NewGraph(p.Name)

	rs := map[string]*dot.Node{}
	for _, r := range p.Spec.Resources {
		n := dot.NewNode(r.Name)
		n.Set("shape", "cylinder")
		rs[r.Name] = n
		g.AddNode(n)
	}

	ts := map[string]*dot.Node{}
	for _, t := range p.Spec.Tasks {
		n := dot.NewNode(t.Name)
		n.Set("shape", "egg")
		ts[t.Name] = n
		g.AddNode(n)

		if t.Resources != nil {
			for _, in := range t.Resources.Inputs {
				rn := rs[in.Resource]
				g.AddEdge(dot.NewEdge(rn, n))
			}
			for _, out := range t.Resources.Outputs {
				rn := rs[out.Resource]
				g.AddEdge(dot.NewEdge(n, rn))
			}
		}

		for _, ra := range t.RunAfter {
			tn := ts[ra]
			e := dot.NewEdge(tn, n)
			e.Set("style", "dashed")
			e.Set("label", "runAfter")
			g.AddEdge(e)
		}
	}

	return g.String(), nil
}
