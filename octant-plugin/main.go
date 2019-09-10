package main // import "github.com/tektoncd/experimental/octant-plugin"

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/pkg/errors"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
	"github.com/tmc/dot"
	"github.com/vmware/octant/pkg/plugin"
	"github.com/vmware/octant/pkg/plugin/service"
	"github.com/vmware/octant/pkg/store"
	"github.com/vmware/octant/pkg/view/component"
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
		},
		service.WithPrinter(handlePrint),
		service.WithTabPrinter(handleTabPrint),
	)
	if err != nil {
		log.Fatal(err)
	}
	p.Serve()
}

// handlePrint is called when Octant wants to print an object.
func handlePrint(request *service.PrintRequest) (plugin.PrintResponse, error) {
	key, err := store.KeyFromObject(request.Object)
	if err != nil {
		return plugin.PrintResponse{}, err
	}
	u, err := request.DashboardClient.Get(request.Context(), key)
	if err != nil {
		return plugin.PrintResponse{}, err
	}

	switch request.Object.GetObjectKind().GroupVersionKind() {
	case taskRunGVK:
		var tr v1alpha1.TaskRun
		if err := duck.FromUnstructured(u, &tr); err != nil {
			return plugin.PrintResponse{}, nil
		}

		resp := plugin.PrintResponse{}
		if tr.Status.PodName != "" {
			// TODO: this 404s for timed-out or cancelled TaskRuns, since we delete the Pod...
			s := tr.Status.PodName
			ref := "../../../workloads/pods/" + tr.Status.PodName
			resp.Status = append(resp.Status, component.SummarySection{Header: "Pod", Content: component.NewLink("Pod Name", s, ref)})
		}
		if tr.Spec.TaskRef != nil {
			s := tr.Spec.TaskRef.Name
			ref := "../../tasks.tekton.dev/" + tr.Spec.TaskRef.Name
			// TODO: handle ClusterTask
			resp.Status = append(resp.Status, component.SummarySection{Header: "Task", Content: component.NewLink("Task Name", s, ref)})

		}
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
		return resp, nil
	case pipelineRunGVK:
		var pr v1alpha1.PipelineRun
		if err := duck.FromUnstructured(u, &pr); err != nil {
			return plugin.PrintResponse{}, nil
		}

		resp := plugin.PrintResponse{}

		s := pr.Spec.PipelineRef.Name
		ref := "../../pipelines.tekton.dev/" + pr.Spec.PipelineRef.Name
		resp.Status = append(resp.Status, component.SummarySection{Header: "Pipeline", Content: component.NewLink("Pipeline Name", s, ref)})

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
		return resp, nil

	case taskGVK:
		var t v1alpha1.Task
		if err := duck.FromUnstructured(u, &t); err != nil {
			return plugin.PrintResponse{}, nil
		}
		resp := plugin.PrintResponse{}
		if t.Spec.Inputs != nil {
			if len(t.Spec.Inputs.Resources) != 0 {
				var md string
				for _, r := range t.Spec.Inputs.Resources {
					md += fmt.Sprintf("* `%s` (%s)\n", r.Name, r.Type)
				}
				card := component.NewCard("Input Resources")
				card.SetBody(component.NewMarkdownText(md))
				resp.Items = append(resp.Items, component.FlexLayoutItem{
					Width: component.WidthFull,
					View:  card,
				})
			}
			if len(t.Spec.Inputs.Params) != 0 {
				var md string
				for _, r := range t.Spec.Inputs.Params {
					md += fmt.Sprintf("* `%s` (%s)", r.Name, r.Type)
					if r.Default != nil {
						switch r.Default.Type {
						case v1alpha1.ParamTypeString:
							md += fmt.Sprintf(" (default: _%s_)", r.Default.StringVal)
						case v1alpha1.ParamTypeArray:
							b, _ := json.Marshal(r.Default.ArrayVal)
							md += fmt.Sprintf(" (default: _%s_)", string(b))
						}
					}
					md += "\n"
				}
				card := component.NewCard("Input Params")
				card.SetBody(component.NewMarkdownText(md))
				resp.Items = append(resp.Items, component.FlexLayoutItem{
					Width: component.WidthFull,
					View:  card,
				})
			}
		}
		if t.Spec.Outputs != nil {
			var md string
			for _, r := range t.Spec.Outputs.Resources {
				md += fmt.Sprintf("* `%s` (%s)\n", r.Name, r.Type)
			}
			card := component.NewCard("Output Resources")
			card.SetBody(component.NewMarkdownText(md))
			resp.Items = append(resp.Items, component.FlexLayoutItem{
				Width: component.WidthFull,
				View:  card,
			})
		}
		return resp, nil
	case pipelineGVK:
		var p v1alpha1.Pipeline
		if err := duck.FromUnstructured(u, &p); err != nil {
			return plugin.PrintResponse{}, nil
		}
		resp := plugin.PrintResponse{}
		if len(p.Spec.Params) != 0 {
			var md string
			for _, r := range p.Spec.Params {
				md += fmt.Sprintf("* `%s` (%s)", r.Name, r.Type)
				if r.Default != nil {
					switch r.Default.Type {
					case v1alpha1.ParamTypeString:
						md += fmt.Sprintf(" (default: _%s_)", r.Default.StringVal)
					case v1alpha1.ParamTypeArray:
						b, _ := json.Marshal(r.Default.ArrayVal)
						md += fmt.Sprintf(" (default: _%s_)", string(b))
					}
				}
				md += "\n"
			}
			card := component.NewCard("Input Params")
			card.SetBody(component.NewMarkdownText(md))
			resp.Items = append(resp.Items, component.FlexLayoutItem{
				Width: component.WidthFull,
				View:  card,
			})
		}
		if len(p.Spec.Resources) != 0 {
			var md string
			for _, r := range p.Spec.Resources {
				md += fmt.Sprintf("* `%s` (%s)\n", r.Name, r.Type)
			}
			card := component.NewCard("Input Resources")
			card.SetBody(component.NewMarkdownText(md))
			resp.Items = append(resp.Items, component.FlexLayoutItem{
				Width: component.WidthFull,
				View:  card,
			})
		}
		if len(p.Spec.Tasks) != 0 {
			var md string
			for _, t := range p.Spec.Tasks {
				ref := "/#/content/overview/namespace/default/custom-resources/tasks.tekton.dev/" + t.TaskRef.Name // TODO: handle ClusterTask
				md += fmt.Sprintf("* [`%s`](%s)\n", t.Name, ref)
			}
			card := component.NewCard("Tasks")
			card.SetBody(component.NewMarkdownText(md))
			resp.Items = append(resp.Items, component.FlexLayoutItem{
				Width: component.WidthFull,
				View:  card,
			})
		}
		return resp, nil
	default:
		return plugin.PrintResponse{}, errors.Errorf("unsupported object %q", request.Object.GetObjectKind().GroupVersionKind())
	}
}

// handleTabPrint is called when Octant wants to print a tab of content.
func handleTabPrint(request *service.PrintRequest) (plugin.TabResponse, error) {
	key, err := store.KeyFromObject(request.Object)
	if err != nil {
		return plugin.TabResponse{}, err
	}
	u, err := request.DashboardClient.Get(request.Context(), key)
	if err != nil {
		return plugin.TabResponse{}, err
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
