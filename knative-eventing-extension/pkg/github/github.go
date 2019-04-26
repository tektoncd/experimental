package github

import (
	"bytes"
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"

	cloudevents "github.com/cloudevents/sdk-go"
	gh "github.com/google/go-github/github"
	tkn "github.com/tektoncd/pipeline/pkg/client/clientset/versioned/typed/pipeline/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}

const (
	pipelinePath      = ".tekton/pipeline.yaml"
	pipelineRunPath   = ".tekton/pipelinerun.yaml"
	pipelineGitResPre = "gh"
	resGitType        = "git"
)

// Trigger has clients needed for fulfiling trigger events
type Trigger struct {
	ghClient     *gh.Client
	tektonClient *tkn.TektonV1alpha1Client
}

// NewTrigger instantiates a new trigger service
func NewTrigger(ghClient *gh.Client, tektonClient *tkn.TektonV1alpha1Client) *Trigger {
	return &Trigger{ghClient, tektonClient}
}

// Handler handles the trigger events and create appropriates pipelineruns
func (t *Trigger) Handler(ctx context.Context, event cloudevents.Event) error {
	switch event.Type() {
	case "com.github.push":
		pe := &gh.PushEvent{}
		if err := event.DataAs(pe); err != nil {
			log.Printf("failed to get push event as PushEvent: %v", err.Error())
			return err
		}
		if pe.Repo == nil || pe.Repo.Owner.Name == nil || pe.Repo.Name == nil {
			log.Printf("Incomplete repo information: %v", pe.Repo)
			return errors.New("incomplete repo info")
		}

		pipeline, err := t.getPipeline(ctx, pe.Repo)
		if err != nil {
			log.Printf("Error Getting Pipeline: %v", err)
			return err
		}
		pipelineRun, err := t.getPipelineRun(ctx, pe.Repo)
		if err != nil {
			log.Printf("Error Getting Pipeline: %v", err)
			return err
		}
		err = t.createPipelineRun(pipeline, pipelineRun, pe)
		if err != nil {
			log.Printf("Error creating pipeline: %v", err)
			return err
		}
	}

	cloudevents.HTTPTransportContextFrom(ctx)
	return nil
}

func (t *Trigger) getPipeline(ctx context.Context, repo *gh.PushEventRepository) (*v1alpha1.Pipeline, error) {
	content, err := t.getGHFileContent(ctx, repo, pipelinePath)
	pipeline := &v1alpha1.Pipeline{}
	err = yaml.NewYAMLToJSONDecoder(bytes.NewReader([]byte(*content))).Decode(pipeline)
	if err != nil {
		log.Fatalf("cannot unmarshal pipeline file: %v", err)
		return nil, err
	}

	return pipeline, err
}

func (t *Trigger) getPipelineRun(ctx context.Context, repo *gh.PushEventRepository) (*v1alpha1.PipelineRun, error) {
	content, err := t.getGHFileContent(ctx, repo, pipelineRunPath)

	pipelineRun := &v1alpha1.PipelineRun{}
	err = yaml.NewYAMLToJSONDecoder(bytes.NewReader([]byte(*content))).Decode(pipelineRun)
	if err != nil {
		log.Fatalf("cannot unmarshal pipeline file: %v", err)
		return nil, err
	}

	log.Printf("prun %+v", pipelineRun)
	return pipelineRun, err
}

func (t *Trigger) getGHFileContent(ctx context.Context, repo *gh.PushEventRepository, path string) (*string, error) {
	fileContents, _, _, err := t.ghClient.Repositories.GetContents(ctx, *repo.Owner.Name, *repo.Name, path, &gh.RepositoryContentGetOptions{})
	if err != nil {
		log.Printf("Repositories.GetContents returned error: %v", err)
		return nil, err
	}

	content, err := fileContents.GetContent()
	if err != nil {
		log.Printf("fail to get content for %v, err: %v", path, err)
		return nil, err
	}
	return &content, nil
}

func (t *Trigger) createPipelineRun(pipeline *v1alpha1.Pipeline,
	pipelinerun *v1alpha1.PipelineRun, pe *gh.PushEvent) error {
	if pipeline.Namespace == "" {
		pipeline.Namespace = "default"
	}
	pipelines := t.tektonClient.Pipelines(pipeline.Namespace)
	oldPipeline, err := pipelines.Get(pipeline.Name, metav1.GetOptions{})
	if err != nil {
		log.Printf("fail to get pipeline: %v", err)
		if pipeline, err = pipelines.Create(pipeline); err != nil {
			log.Printf("fail to create pipeline: pipeline:%v,err:%v", pipeline, err)
			return err
		}
	} else {
		pipeline.ResourceVersion = oldPipeline.ResourceVersion
		if _, err = pipelines.Update(pipeline); err != nil {
			log.Printf("fail to update pipeline: %v", err)
			return err
		}
	}

	repoResName := pipelineGitResPre + "-" + *pe.Repo.Owner.Name +
		"-" + *pe.Repo.Name
	res := t.tektonClient.PipelineResources(pipeline.Namespace)
	updatedGitRes := &v1alpha1.PipelineResource{
		ObjectMeta: metav1.ObjectMeta{Name: repoResName, Namespace: pipeline.Namespace},
		Spec: v1alpha1.PipelineResourceSpec{
			Type: resGitType,
			Params: []v1alpha1.Param{{Name: "revision", Value: *pe.HeadCommit.ID},
				{Name: "url", Value: *pe.Repo.URL}},
		},
	}

	r, err := res.Get(repoResName, metav1.GetOptions{})
	if err != nil {
		if _, err = res.Create(updatedGitRes); err != nil {
			log.Printf("fail to create resource: %v", err)
			return err
		}
	} else {
		updatedGitRes.ResourceVersion = r.ResourceVersion
		if _, err = res.Update(updatedGitRes); err != nil {
			log.Printf("fail to update resource: %v", err)
			return err
		}
	}

	log.Printf("pipelinerun :%+v", pipelinerun)
	pruns := t.tektonClient.PipelineRuns(pipeline.Namespace)
	pipelinerun.Name = pipelinerun.Name +
		strconv.FormatInt(time.Now().UTC().UnixNano(), 10)
	_, err = pruns.Create(pipelinerun)
	if err != nil {
		log.Printf("fail to create pipelinerun: %v", err)
		return err
	}
	return nil
}
