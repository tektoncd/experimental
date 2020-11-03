# tekton-pipeline
Python SDK for Tekton Pipeline

## Requirements.

Python 2.7 and 3.4+

## Installation & Usage
### pip install

Tekton Pipeline Python SDK can be installed by `pip` or `Setuptools`.

```sh
pip install tekton-pipeline
```

### Setuptools

Install via [Setuptools](http://pypi.python.org/pypi/setuptools).

```sh
python setup.py install --user
```
(or `sudo python setup.py install` to install the package for all users)

## Getting Started

Please follow the [installation procedure](#installation--usage) and then run the [examples](examples/taskrun.ipynb)


## Documentation for API Endpoints

Class | Method | Description
------------ | ------------- | -------------
[TektonClient](docs/TektonClient.md) | [create](docs/TektonClient.md#create) | Create Tekton object|
[TektonClient](docs/TektonClient.md) | [get](docs/TektonClient.md#get) | get Tekton object|
[TektonClient](docs/TektonClient.md) | [delete](docs/TektonClient.md#delete) | delete Tekton object|

## Documentation For Models

 - [PodTemplate](docs/PodTemplate.md)
 - [V1alpha1PipelineResource](docs/V1alpha1PipelineResource.md)
 - [V1alpha1PipelineResourceList](docs/V1alpha1PipelineResourceList.md)
 - [V1alpha1PipelineResourceSpec](docs/V1alpha1PipelineResourceSpec.md)
 - [V1alpha1ResourceDeclaration](docs/V1alpha1ResourceDeclaration.md)
 - [V1alpha1ResourceParam](docs/V1alpha1ResourceParam.md)
 - [V1alpha1SecretParam](docs/V1alpha1SecretParam.md)
 - [V1beta1ArrayOrString](docs/V1beta1ArrayOrString.md)
 - [V1beta1CannotConvertError](docs/V1beta1CannotConvertError.md)
 - [V1beta1CloudEventDelivery](docs/V1beta1CloudEventDelivery.md)
 - [V1beta1CloudEventDeliveryState](docs/V1beta1CloudEventDeliveryState.md)
 - [V1beta1ClusterTask](docs/V1beta1ClusterTask.md)
 - [V1beta1ClusterTaskList](docs/V1beta1ClusterTaskList.md)
 - [V1beta1ConditionCheck](docs/V1beta1ConditionCheck.md)
 - [V1beta1ConditionCheckStatus](docs/V1beta1ConditionCheckStatus.md)
 - [V1beta1ConditionCheckStatusFields](docs/V1beta1ConditionCheckStatusFields.md)
 - [V1beta1EmbeddedTask](docs/V1beta1EmbeddedTask.md)
 - [V1beta1InternalTaskModifier](docs/V1beta1InternalTaskModifier.md)
 - [V1beta1Param](docs/V1beta1Param.md)
 - [V1beta1ParamSpec](docs/V1beta1ParamSpec.md)
 - [V1beta1Pipeline](docs/V1beta1Pipeline.md)
 - [V1beta1PipelineDeclaredResource](docs/V1beta1PipelineDeclaredResource.md)
 - [V1beta1PipelineList](docs/V1beta1PipelineList.md)
 - [V1beta1PipelineRef](docs/V1beta1PipelineRef.md)
 - [V1beta1PipelineResourceBinding](docs/V1beta1PipelineResourceBinding.md)
 - [V1beta1PipelineResourceRef](docs/V1beta1PipelineResourceRef.md)
 - [V1beta1PipelineResourceResult](docs/V1beta1PipelineResourceResult.md)
 - [V1beta1PipelineResult](docs/V1beta1PipelineResult.md)
 - [V1beta1PipelineRun](docs/V1beta1PipelineRun.md)
 - [V1beta1PipelineRunConditionCheckStatus](docs/V1beta1PipelineRunConditionCheckStatus.md)
 - [V1beta1PipelineRunList](docs/V1beta1PipelineRunList.md)
 - [V1beta1PipelineRunResult](docs/V1beta1PipelineRunResult.md)
 - [V1beta1PipelineRunSpec](docs/V1beta1PipelineRunSpec.md)
 - [V1beta1PipelineRunSpecServiceAccountName](docs/V1beta1PipelineRunSpecServiceAccountName.md)
 - [V1beta1PipelineRunStatus](docs/V1beta1PipelineRunStatus.md)
 - [V1beta1PipelineRunStatusFields](docs/V1beta1PipelineRunStatusFields.md)
 - [V1beta1PipelineRunTaskRunStatus](docs/V1beta1PipelineRunTaskRunStatus.md)
 - [V1beta1PipelineSpec](docs/V1beta1PipelineSpec.md)
 - [V1beta1PipelineTask](docs/V1beta1PipelineTask.md)
 - [V1beta1PipelineTaskCondition](docs/V1beta1PipelineTaskCondition.md)
 - [V1beta1PipelineTaskInputResource](docs/V1beta1PipelineTaskInputResource.md)
 - [V1beta1PipelineTaskMetadata](docs/V1beta1PipelineTaskMetadata.md)
 - [V1beta1PipelineTaskOutputResource](docs/V1beta1PipelineTaskOutputResource.md)
 - [V1beta1PipelineTaskParam](docs/V1beta1PipelineTaskParam.md)
 - [V1beta1PipelineTaskResources](docs/V1beta1PipelineTaskResources.md)
 - [V1beta1PipelineTaskRun](docs/V1beta1PipelineTaskRun.md)
 - [V1beta1PipelineTaskRunSpec](docs/V1beta1PipelineTaskRunSpec.md)
 - [V1beta1PipelineWorkspaceDeclaration](docs/V1beta1PipelineWorkspaceDeclaration.md)
 - [V1beta1ResultRef](docs/V1beta1ResultRef.md)
 - [V1beta1Sidecar](docs/V1beta1Sidecar.md)
 - [V1beta1SidecarState](docs/V1beta1SidecarState.md)
 - [V1beta1SkippedTask](docs/V1beta1SkippedTask.md)
 - [V1beta1Step](docs/V1beta1Step.md)
 - [V1beta1StepState](docs/V1beta1StepState.md)
 - [V1beta1Task](docs/V1beta1Task.md)
 - [V1beta1TaskList](docs/V1beta1TaskList.md)
 - [V1beta1TaskRef](docs/V1beta1TaskRef.md)
 - [V1beta1TaskResource](docs/V1beta1TaskResource.md)
 - [V1beta1TaskResourceBinding](docs/V1beta1TaskResourceBinding.md)
 - [V1beta1TaskResources](docs/V1beta1TaskResources.md)
 - [V1beta1TaskResult](docs/V1beta1TaskResult.md)
 - [V1beta1TaskRun](docs/V1beta1TaskRun.md)
 - [V1beta1TaskRunInputs](docs/V1beta1TaskRunInputs.md)
 - [V1beta1TaskRunList](docs/V1beta1TaskRunList.md)
 - [V1beta1TaskRunOutputs](docs/V1beta1TaskRunOutputs.md)
 - [V1beta1TaskRunResources](docs/V1beta1TaskRunResources.md)
 - [V1beta1TaskRunResult](docs/V1beta1TaskRunResult.md)
 - [V1beta1TaskRunSpec](docs/V1beta1TaskRunSpec.md)
 - [V1beta1TaskRunStatus](docs/V1beta1TaskRunStatus.md)
 - [V1beta1TaskRunStatusFields](docs/V1beta1TaskRunStatusFields.md)
 - [V1beta1TaskSpec](docs/V1beta1TaskSpec.md)
 - [V1beta1WhenExpression](docs/V1beta1WhenExpression.md)
 - [V1beta1WorkspaceBinding](docs/V1beta1WorkspaceBinding.md)
 - [V1beta1WorkspaceDeclaration](docs/V1beta1WorkspaceDeclaration.md)
 - [V1beta1WorkspacePipelineTaskBinding](docs/V1beta1WorkspacePipelineTaskBinding.md)


