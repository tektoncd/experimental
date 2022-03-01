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
	"fmt"

	cdeevents "github.com/cdfoundation/sig-events/cde/sdk/go/pkg/cdf/events"
	cloudevents "github.com/cloudevents/sdk-go/v2"
	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	"knative.dev/pkg/apis"
)

var artifactMappings = map[string]resultMapping{
	"artifactId": {
		defaultResultName:       "cd.artifact.id",
		annotationResultNameKey: "cd.events/results.artifact.id",
	},
	"artifactName": {
		defaultResultName:       "cd.artifact.name",
		annotationResultNameKey: "cd.events/results.artifact.name",
	},
	"artifactVersion": {
		defaultResultName:       "cd.artifact.version",
		annotationResultNameKey: "cd.events/results.artifact.version",
	},
}

const ArtifactPackagedEventAnnotation cdEventAnnotationType = "cd.artifact.packaged"
const ArtifactPublishedEventAnnotation cdEventAnnotationType = "cd.artifact.published"

// getArtifactEventType returns an eventType is conditions are met
func getArtifactEventType(runObject objectWithCondition, cdEventType cdeevents.CDEventType) (*EventType, error) {
	c := runObject.GetStatusCondition().GetCondition(apis.ConditionSucceeded)
	if c == nil {
		return nil, fmt.Errorf("no condition for ConditionSucceeded in %T", runObject)
	}
	if !c.IsTrue() {
		return nil, fmt.Errorf("no artifact event for condition %T", c)
	}
	eventType := &EventType{
		Type: cdEventType,
	}
	switch runObject.(type) {
	case *v1beta1.TaskRun, *v1beta1.PipelineRun:
		return eventType, nil
	}
	return nil, fmt.Errorf("unknown type of Tekton resource")
}

// getArtifactPackagedEventType returns a CDF Artifact Packaged EventType if objectWithCondition meets conditions
func getArtifactPackagedEventType(runObject objectWithCondition) (*EventType, error) {
	annotations := runObject.GetObjectMeta().GetAnnotations()
	if _, ok := annotations[ArtifactPackagedEventAnnotation.String()]; ok {
		return getArtifactEventType(runObject, cdeevents.ArtifactPackagedEventV1)
	}
	return nil, fmt.Errorf("no %s annotation found", ArtifactPackagedEventAnnotation.String())
}

// getArtifactPublishedEventType returns a CDF Artifact Published EventType if objectWithCondition meets conditions
func getArtifactPublishedEventType(runObject objectWithCondition) (*EventType, error) {
	annotations := runObject.GetObjectMeta().GetAnnotations()
	if _, ok := annotations[ArtifactPublishedEventAnnotation.String()]; ok {
		return getArtifactEventType(runObject, cdeevents.ArtifactPublishedEventV1)
	}
	return nil, fmt.Errorf("no %s annotation found", ArtifactPublishedEventAnnotation.String())
}

// getArtifactEventData
func getArtifactEventData(runObject objectWithCondition) (CDECloudEventData, error) {
	cdeCloudEventData, err := getEventData(runObject)
	if err != nil {
		return nil, err
	}
	for mappingName, mapping := range artifactMappings {
		mappingValue, err := resultForMapping(runObject, mapping)
		if err != nil {
			return nil, err
		}
		cdeCloudEventData[mappingName] = mappingValue
	}
	return cdeCloudEventData, nil
}

// getArtifactPackagedEventData returns the data for a CDF Artifact Packaged Event
func getArtifactPackagedEventData(runObject objectWithCondition) (CDECloudEventData, error) {
	return getArtifactEventData(runObject)
}

// getArtifactPublishedEventData returns the data for a CDF Artifact Packaged Event
func getArtifactPublishedEventData(runObject objectWithCondition) (CDECloudEventData, error) {
	return getArtifactEventData(runObject)
}

func artifactPackagedEventForObjectWithCondition(runObject objectWithCondition) (*cloudevents.Event, error) {
	etype, err := getArtifactPackagedEventType(runObject)
	if err != nil {
		return nil, err
	}
	data, err := getArtifactPackagedEventData(runObject)
	if err != nil {
		return nil, err
	}
	params := cdeevents.ArtifactEventParams{
		ArtifactName:    data["artifactName"],
		ArtifactVersion: data["artifactVersion"],
		ArtifactId:      data["artifactId"],
		ArtifactData:    data,
	}
	event, err := cdeevents.CreateArtifactEvent(etype.Type, params)
	if err != nil {
		return nil, err
	}
	event.SetSubject(runObject.GetObjectMeta().GetName())
	event.SetSource(getSource(runObject))
	return &event, nil
}

func artifactPublishedEventForObjectWithCondition(runObject objectWithCondition) (*cloudevents.Event, error) {
	etype, err := getArtifactPublishedEventType(runObject)
	if err != nil {
		return nil, err
	}
	data, err := getArtifactPublishedEventData(runObject)
	if err != nil {
		return nil, err
	}
	params := cdeevents.ArtifactEventParams{
		ArtifactName:    data["artifactName"],
		ArtifactVersion: data["artifactVersion"],
		ArtifactId:      data["artifactId"],
		ArtifactData:    data,
	}
	event, err := cdeevents.CreateArtifactEvent(etype.Type, params)
	if err != nil {
		return nil, err
	}
	event.SetSubject(runObject.GetObjectMeta().GetName())
	event.SetSource(getSource(runObject))
	return &event, nil
}
