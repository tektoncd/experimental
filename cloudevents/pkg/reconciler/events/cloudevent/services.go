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

var (
	serviceMappings = map[string]resultMapping{
		"serviceEnvId": {
			defaultResultName:       "cd.service.envId",
			annotationResultNameKey: "cd.events/results.service.envid",
		},
		"serviceName": {
			defaultResultName:       "cd.service.name",
			annotationResultNameKey: "cd.events/results.service.name",
		},
		"serviceVersion": {
			defaultResultName:       "cd.service.version",
			annotationResultNameKey: "cd.events/results.service.version",
		},
	}
)

const ServiceDeployedEventAnnotation cdEventAnnotationType = "cd.service.deployed"
const ServiceRolledbackEventAnnotation cdEventAnnotationType = "cd.service.rolledback"
const ServiceUpgradedEventAnnotation cdEventAnnotationType = "cd.service.upgraded"
const ServiceRemovedEventAnnotation cdEventAnnotationType = "cd.service.removed"

// getServiceEventType returns an eventType is conditions are met
func getServiceEventType(runObject objectWithCondition, cdEventType cdeevents.CDEventType) (*EventType, error) {
	c := runObject.GetStatusCondition().GetCondition(apis.ConditionSucceeded)
	if c == nil {
		return nil, fmt.Errorf("no condition for ConditionSucceeded in %T", runObject)
	}
	if !c.IsTrue() {
		return nil, fmt.Errorf("no service event for condition %T", c)
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

// getServiceDeployedEventType returns a CDF Service Deployed EventType if objectWithCondition meets conditions
func getServiceDeployedEventType(runObject objectWithCondition) (*EventType, error) {
	annotations := runObject.GetObjectMeta().GetAnnotations()
	if _, ok := annotations[ServiceDeployedEventAnnotation.String()]; ok {
		return getServiceEventType(runObject, cdeevents.ServiceDeployedEventV1)
	}
	return nil, fmt.Errorf("no %s annotation found", ServiceDeployedEventAnnotation.String())
}

// getServiceRolledbackEventType returns a CDF Service Rolledback EventType if objectWithCondition meets conditions
func getServiceRolledbackEventType(runObject objectWithCondition) (*EventType, error) {
	annotations := runObject.GetObjectMeta().GetAnnotations()
	if _, ok := annotations[ServiceRolledbackEventAnnotation.String()]; ok {
		return getServiceEventType(runObject, cdeevents.ServiceRolledbackEventV1)
	}
	return nil, fmt.Errorf("no %s annotation found", ServiceRolledbackEventAnnotation.String())
}

// getServiceUpgradedEventType returns a CDF Service Upgraded EventType if objectWithCondition meets conditions
func getServiceUpgradedEventType(runObject objectWithCondition) (*EventType, error) {
	annotations := runObject.GetObjectMeta().GetAnnotations()
	if _, ok := annotations[ServiceUpgradedEventAnnotation.String()]; ok {
		return getServiceEventType(runObject, cdeevents.ServiceUpgradedEventV1)
	}
	return nil, fmt.Errorf("no %s annotation found", ServiceUpgradedEventAnnotation.String())
}

// getServiceRemovedEventType returns a CDF Service Removed EventType if objectWithCondition meets conditions
func getServiceRemovedEventType(runObject objectWithCondition) (*EventType, error) {
	annotations := runObject.GetObjectMeta().GetAnnotations()
	if _, ok := annotations[ServiceRemovedEventAnnotation.String()]; ok {
		return getServiceEventType(runObject, cdeevents.ServiceRemovedEventV1)
	}
	return nil, fmt.Errorf("no %s annotation found", ServiceRemovedEventAnnotation.String())
}

// getServiceEventData
func getServiceEventData(runObject objectWithCondition) (CDECloudEventData, error) {
	cdeCloudEventData, err := getEventData(runObject)
	if err != nil {
		return nil, err
	}
	for mappingName, mapping := range serviceMappings {
		mappingValue, err := resultForMapping(runObject, mapping)
		if err != nil {
			return nil, err
		}
		cdeCloudEventData[mappingName] = mappingValue
	}
	return cdeCloudEventData, nil
}

func serviceEventFromType(eventType *EventType, runObject objectWithCondition) (*cloudevents.Event, error) {
	data, err := getServiceEventData(runObject)
	if err != nil {
		return nil, err
	}
	event, err := cdeevents.CreateServiceEvent(eventType.Type,
		data["serviceEnvId"], data["serviceName"], data["serviceVersion"], data)
	if err != nil {
		return nil, err
	}
	event.SetSubject(runObject.GetObjectMeta().GetName())
	event.SetSource(getSource(runObject))
	return &event, nil
}

func serviceDeployedEventForObjectWithCondition(runObject objectWithCondition) (*cloudevents.Event, error) {
	etype, err := getServiceDeployedEventType(runObject)
	if err != nil {
		return nil, err
	}
	return serviceEventFromType(etype, runObject)
}

func serviceRolledbackEventForObjectWithCondition(runObject objectWithCondition) (*cloudevents.Event, error) {
	etype, err := getServiceRolledbackEventType(runObject)
	if err != nil {
		return nil, err
	}
	return serviceEventFromType(etype, runObject)
}

func serviceUpgradedEventForObjectWithCondition(runObject objectWithCondition) (*cloudevents.Event, error) {
	etype, err := getServiceUpgradedEventType(runObject)
	if err != nil {
		return nil, err
	}
	return serviceEventFromType(etype, runObject)
}

func serviceRemovedEventForObjectWithCondition(runObject objectWithCondition) (*cloudevents.Event, error) {
	etype, err := getServiceRemovedEventType(runObject)
	if err != nil {
		return nil, err
	}
	return serviceEventFromType(etype, runObject)
}
