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

package cloudevent

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/tektoncd/pipeline/test/diff"
	"testing"

	cdeevents "github.com/cdfoundation/sig-events/cde/sdk/go/pkg/cdf/events"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"github.com/tektoncd/pipeline/test/names"
	corev1 "k8s.io/api/core/v1"
)

func getTaskRunByConditionAndResults(status corev1.ConditionStatus, reason string, annotations map[string]string, results map[string]string) *v1beta1.TaskRun {
	taskRun := getTaskRunByCondition(status, reason)
	taskRunResults := []v1beta1.TaskRunResult{}
	for key, value := range results {
		taskRunResults = append(taskRunResults, v1beta1.TaskRunResult{
			Name:  key,
			Value: value,
		})
	}
	taskRun.Status.TaskRunResults = taskRunResults
	taskRun.ObjectMeta.Annotations = annotations
	return taskRun
}

func getPipelineRunByConditionAndResults(status corev1.ConditionStatus, reason string, annotations map[string]string, results map[string]string) *v1beta1.PipelineRun {
	pipelineRun := getPipelineRunByCondition(status, reason)
	pipelineRunResults := []v1beta1.PipelineRunResult{}
	for key, value := range results {
		pipelineRunResults = append(pipelineRunResults, v1beta1.PipelineRunResult{
			Name:  key,
			Value: value,
		})
	}
	pipelineRun.Status.PipelineResults = pipelineRunResults
	pipelineRun.ObjectMeta.Annotations = annotations
	return pipelineRun
}

var (
	artifactPackagedEventType  = &EventType{Type: cdeevents.ArtifactPackagedEventV1}
	artifactPublishedEventType = &EventType{Type: cdeevents.ArtifactPublishedEventV1}
)

func TestArtifactEventsForTaskRun(t *testing.T) {
	taskRunTests := []struct {
		desc      string
		taskRun   *v1beta1.TaskRun
		wantError bool
	}{{
		desc:      "taskrun with no annotations",
		taskRun:   getTaskRunByCondition(corev1.ConditionUnknown, v1beta1.TaskRunReasonStarted.String()),
		wantError: true,
	}, {
		desc: "taskrun with annotation, started",
		taskRun: getTaskRunByConditionAndResults(
			corev1.ConditionUnknown,
			v1beta1.TaskRunReasonStarted.String(),
			map[string]string{ArtifactPackagedEventAnnotation.String(): ""},
			map[string]string{}),
		wantError: true,
	}, {
		desc: "taskrun with annotation, finished, failed",
		taskRun: getTaskRunByConditionAndResults(
			corev1.ConditionFalse,
			"meh",
			map[string]string{ArtifactPackagedEventAnnotation.String(): ""},
			map[string]string{}),
		wantError: true,
	}, {
		desc: "taskrun with annotation, finished, succeeded",
		taskRun: getTaskRunByConditionAndResults(
			corev1.ConditionTrue,
			"yay",
			map[string]string{ArtifactPackagedEventAnnotation.String(): ""},
			map[string]string{}),
		wantError: false,
	}}

	for _, c := range taskRunTests {
		t.Run(c.desc, func(t *testing.T) {
			names.TestingSeed()

			got, err := getArtifactPackagedEventType(c.taskRun)
			if err != nil {
				if !c.wantError {
					t.Fatalf("I did not expect an error but I got %s", err)
				}
			} else {
				if c.wantError {
					t.Fatalf("I did expect an error but I got %s", got)
				}
				if d := cmp.Diff(artifactPackagedEventType, got); d != "" {
					t.Errorf("Wrong Event Type %s", diff.PrintWantGot(d))
				}
			}
		})
	}
}

func TestArtifactEventsForPipelineRun(t *testing.T) {
	pipelineRunTests := []struct {
		desc        string
		pipelineRun *v1beta1.PipelineRun
		wantError   bool
	}{{
		desc:        "pipelinerun with no annotations",
		pipelineRun: getPipelineRunByCondition(corev1.ConditionUnknown, v1beta1.PipelineRunReasonStarted.String()),
		wantError:   true,
	}, {
		desc: "pipelinerun with annotation, started",
		pipelineRun: getPipelineRunByConditionAndResults(
			corev1.ConditionUnknown,
			v1beta1.PipelineRunReasonStarted.String(),
			map[string]string{ArtifactPublishedEventAnnotation.String(): ""},
			map[string]string{}),
		wantError: true,
	}, {
		desc: "pipelinerun with annotation, finished, failed",
		pipelineRun: getPipelineRunByConditionAndResults(
			corev1.ConditionFalse,
			"meh",
			map[string]string{ArtifactPublishedEventAnnotation.String(): ""},
			map[string]string{}),
		wantError: true,
	}, {
		desc: "pipelinerun with annotation, finished, succeeded",
		pipelineRun: getPipelineRunByConditionAndResults(
			corev1.ConditionTrue,
			"yay",
			map[string]string{ArtifactPublishedEventAnnotation.String(): ""},
			map[string]string{}),
		wantError: false,
	}}

	for _, c := range pipelineRunTests {
		t.Run(c.desc, func(t *testing.T) {
			names.TestingSeed()

			got, err := getArtifactPublishedEventType(c.pipelineRun)
			if err != nil {
				if !c.wantError {
					t.Fatalf("I did not expect an error but I got %s", err)
				}
			} else {
				if c.wantError {
					t.Fatalf("I did expect an error but I got %s", got)
				}
				if d := cmp.Diff(artifactPublishedEventType, got); d != "" {
					t.Errorf("Wrong Event Type %s", diff.PrintWantGot(d))
				}
			}
		})
	}
}

func TestGetArtifactEventDataPipelineRun(t *testing.T) {
	pipelineRunTests := []struct {
		desc        string
		pipelineRun *v1beta1.PipelineRun
		wantData    CDECloudEventData
		wantError   bool
	}{{
		desc: "pipelinerun with default results, all",
		pipelineRun: getPipelineRunByConditionAndResults(
			corev1.ConditionUnknown,
			v1beta1.PipelineRunReasonStarted.String(),
			map[string]string{ArtifactPublishedEventAnnotation.String(): ""},
			map[string]string{
				"cd.artifact.id":      "test123",
				"cd.artifact.version": "v123",
				"cd.artifact.name":    "testArtifact",
			}),
		wantData: CDECloudEventData{
			"artifactId":      "test123",
			"artifactVersion": "v123",
			"artifactName":    "testArtifact"},
		wantError: false,
	}, {
		desc: "pipelinerun with default results, missing",
		pipelineRun: getPipelineRunByConditionAndResults(
			corev1.ConditionUnknown,
			v1beta1.PipelineRunReasonStarted.String(),
			map[string]string{ArtifactPublishedEventAnnotation.String(): ""},
			map[string]string{
				"cd.artifact.id":      "test123",
				"cd.artifact.version": "v123",
			}),
		wantData:  nil,
		wantError: true,
	}, {
		desc: "pipelinerun with overwritten results, all",
		pipelineRun: getPipelineRunByConditionAndResults(
			corev1.ConditionUnknown,
			v1beta1.PipelineRunReasonStarted.String(),
			map[string]string{
				ArtifactPublishedEventAnnotation.String():           "",
				mappings["artifactId"].annotationResultNameKey:      "builtImage",
				mappings["artifactVersion"].annotationResultNameKey: "tag"},
			map[string]string{
				"builtImage":       "test123",
				"cd.artifact.name": "testimage",
				"tag":              "v123",
			}),
		wantData: CDECloudEventData{
			"artifactId":      "test123",
			"artifactVersion": "v123",
			"artifactName":    "testimage"},
		wantError: false,
	}, {
		desc: "pipelinerun with overwritten results, missing an overwritten one",
		pipelineRun: getPipelineRunByConditionAndResults(
			corev1.ConditionUnknown,
			v1beta1.PipelineRunReasonStarted.String(),
			map[string]string{
				ArtifactPublishedEventAnnotation.String():           "",
				mappings["artifactId"].annotationResultNameKey:      "builtImage",
				mappings["artifactVersion"].annotationResultNameKey: "tag"},
			map[string]string{
				"builtImage":       "test123",
				"cd.artifact.name": "testimage",
			}),
		wantData:  nil,
		wantError: true,
	}}

	for _, c := range pipelineRunTests {
		t.Run(c.desc, func(t *testing.T) {
			names.TestingSeed()

			got, err := getArtifactEventData(c.pipelineRun)
			if err != nil {
				if !c.wantError {
					t.Fatalf("I did not expect an error but I got %s", err)
				}
			} else {
				if c.wantError {
					t.Fatalf("I did expect an error but I got %s", got)
				}
				opt := cmpopts.IgnoreMapEntries(func(k string, v string) bool { return k == "pipelinerun" })
				if d := cmp.Diff(c.wantData, got, opt); d != "" {
					t.Errorf("Wrong Event Data %s", diff.PrintWantGot(d))
				}
			}
		})
	}
}

func TestGetArtifactEventDataTaskRun(t *testing.T) {
	taskRunTests := []struct {
		desc      string
		taskRun   *v1beta1.TaskRun
		wantData  CDECloudEventData
		wantError bool
	}{{
		desc: "taskrun with default results, all",
		taskRun: getTaskRunByConditionAndResults(
			corev1.ConditionUnknown,
			v1beta1.TaskRunReasonStarted.String(),
			map[string]string{ArtifactPublishedEventAnnotation.String(): ""},
			map[string]string{
				"cd.artifact.id":      "test123",
				"cd.artifact.version": "v123",
				"cd.artifact.name":    "testArtifact",
			}),
		wantData: CDECloudEventData{
			"artifactId":      "test123",
			"artifactVersion": "v123",
			"artifactName":    "testArtifact"},
		wantError: false,
	}, {
		desc: "taskrun with default results, missing",
		taskRun: getTaskRunByConditionAndResults(
			corev1.ConditionUnknown,
			v1beta1.TaskRunReasonStarted.String(),
			map[string]string{ArtifactPublishedEventAnnotation.String(): ""},
			map[string]string{
				"cd.artifact.id":      "test123",
				"cd.artifact.version": "v123",
			}),
		wantData:  nil,
		wantError: true,
	}, {
		desc: "taskrun with overwritten results, all",
		taskRun: getTaskRunByConditionAndResults(
			corev1.ConditionUnknown,
			v1beta1.TaskRunReasonStarted.String(),
			map[string]string{
				ArtifactPublishedEventAnnotation.String():           "",
				mappings["artifactId"].annotationResultNameKey:      "builtImage",
				mappings["artifactVersion"].annotationResultNameKey: "tag"},
			map[string]string{
				"builtImage":       "test123",
				"cd.artifact.name": "testimage",
				"tag":              "v123",
			}),
		wantData: CDECloudEventData{
			"artifactId":      "test123",
			"artifactVersion": "v123",
			"artifactName":    "testimage"},
		wantError: false,
	}, {
		desc: "taskrun with overwritten results, missing an overwritten one",
		taskRun: getTaskRunByConditionAndResults(
			corev1.ConditionUnknown,
			v1beta1.TaskRunReasonStarted.String(),
			map[string]string{
				ArtifactPublishedEventAnnotation.String():           "",
				mappings["artifactId"].annotationResultNameKey:      "builtImage",
				mappings["artifactVersion"].annotationResultNameKey: "tag"},
			map[string]string{
				"builtImage":       "test123",
				"cd.artifact.name": "testimage",
			}),
		wantData:  nil,
		wantError: true,
	}}

	for _, c := range taskRunTests {
		t.Run(c.desc, func(t *testing.T) {
			names.TestingSeed()

			got, err := getArtifactEventData(c.taskRun)
			if err != nil {
				if !c.wantError {
					t.Fatalf("I did not expect an error but I got %s", err)
				}
			} else {
				if c.wantError {
					t.Fatalf("I did expect an error but I got %s", got)
				}
				opt := cmpopts.IgnoreMapEntries(func(k string, v string) bool { return k == "taskrun" })
				if d := cmp.Diff(c.wantData, got, opt); d != "" {
					t.Errorf("Wrong Event Data %s", diff.PrintWantGot(d))
				}
			}
		})
	}
}

func TestArtifactPublishedEvent(t *testing.T) {
	artifactPublishedTests := []struct {
		desc                string
		object              interface{}
		wantEventExtensions map[string]interface{}
	}{{
		desc: "artifact event for taskrun",
		object: getTaskRunByConditionAndResults(
			corev1.ConditionTrue,
			v1beta1.TaskRunReasonSuccessful.String(),
			map[string]string{ArtifactPublishedEventAnnotation.String(): ""},
			map[string]string{
				"cd.artifact.id":      "test123",
				"cd.artifact.version": "v123",
				"cd.artifact.name":    "testArtifact",
			}),
		wantEventExtensions: map[string]interface{}{
			"artifactid":      "test123",
			"artifactversion": "v123",
			"artifactname":    "testArtifact",
		},
	}, {
		desc: "artifact event for pipelinerun",
		object: getPipelineRunByConditionAndResults(
			corev1.ConditionTrue,
			v1beta1.PipelineRunReasonSuccessful.String(),
			map[string]string{ArtifactPublishedEventAnnotation.String(): ""},
			map[string]string{
				"cd.artifact.id":      "test123",
				"cd.artifact.version": "v123",
				"cd.artifact.name":    "testArtifact",
			}),
		wantEventExtensions: map[string]interface{}{
			"artifactid":      "test123",
			"artifactversion": "v123",
			"artifactname":    "testArtifact",
		},
	}}

	for _, c := range artifactPublishedTests {
		t.Run(c.desc, func(t *testing.T) {
			names.TestingSeed()

			got, err := artifactPublishedEvenForObjectWithCondition(c.object.(objectWithCondition))
			if err != nil {
				t.Fatalf("I did not expect an error but I got %s", err)
			} else {
				extensions := got.Extensions()
				if d := cmp.Diff(c.wantEventExtensions, extensions); d != "" {
					t.Errorf("Wrong Event Extenstions %s", diff.PrintWantGot(d))
				}
			}
		})
	}
}

func TestArtifactPackagedEvent(t *testing.T) {
	artifactPackagedTests := []struct {
		desc                string
		object              interface{}
		wantEventExtensions map[string]interface{}
	}{{
		desc: "artifact event for taskrun",
		object: getTaskRunByConditionAndResults(
			corev1.ConditionTrue,
			v1beta1.TaskRunReasonSuccessful.String(),
			map[string]string{
				ArtifactPackagedEventAnnotation.String(): ""},
			map[string]string{
				"cd.artifact.id":      "test123",
				"cd.artifact.version": "v123",
				"cd.artifact.name":    "testArtifact",
			}),
		wantEventExtensions: map[string]interface{}{
			"artifactid":      "test123",
			"artifactversion": "v123",
			"artifactname":    "testArtifact",
		},
	}, {
		desc: "artifact event for pipelinerun",
		object: getPipelineRunByConditionAndResults(
			corev1.ConditionTrue,
			v1beta1.PipelineRunReasonSuccessful.String(),
			map[string]string{
				ArtifactPackagedEventAnnotation.String(): ""},
			map[string]string{
				"cd.artifact.id":      "test123",
				"cd.artifact.version": "v123",
				"cd.artifact.name":    "testArtifact",
			}),
		wantEventExtensions: map[string]interface{}{
			"artifactid":      "test123",
			"artifactversion": "v123",
			"artifactname":    "testArtifact",
		},
	}}

	for _, c := range artifactPackagedTests {
		t.Run(c.desc, func(t *testing.T) {
			names.TestingSeed()

			got, err := artifactPackagedEvenForObjectWithCondition(c.object.(objectWithCondition))
			if err != nil {
				t.Fatalf("I did not expect an error but I got %s", err)
			} else {
				extensions := got.Extensions()
				if d := cmp.Diff(c.wantEventExtensions, extensions); d != "" {
					t.Errorf("Wrong Event Extenstions %s", diff.PrintWantGot(d))
				}
			}
		})
	}
}
