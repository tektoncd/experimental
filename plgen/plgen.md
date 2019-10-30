# plgen

`plgen` generates tekton pipelines that you can deploy onto kubernetes cluster with minimal or zero changes.

The tool provides a high level abstraction on top of the tekton pipeline semantics, hiding most of the details and the `yaml complexity` altogether. The intent is to drastically improve user experience on working with pipelines, and transforming pipelines as an integral part of kabanero's capabiity.

The main attraction of `plgen` is its input syntax - which it derives from Dockerfile. While Dockerfile uses these verbs to define attributes, execution environment and a sequence of actions that lead upto generation of an image, `plgen` uses those verbs for sequencing discrete steps into a pipeline definition, with meanings of most of the verbs in-tact.

Supported verbs at the moment are:
```
ARG
ARGIN
ARGOUT
FROM
RUN
LABEL
ENV
MOUNT
USER
```

Feel free to raise an issue / rfe in this repo, if there is need to define new verbs.

Here is an example input and the generated pipeline:\s\s
$ cat pl.txt 

```Dockerfile
LABEL targz
FROM ubuntu
USER kubernetes-user
MOUNT containers=/var/lib/containers
ARG input=http://example.com/archive.tar.gz
ENV foo=bar
RUN tar xzvf $input
RUN cat source/file.txt
```

$ plgen pl.txt
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kubernetes-user
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: admin
subjects:
  kind: ServiceAccount
  name: kubernetes-user
  namespace: default
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
---
apiVersion: v1
items:
- apiVersion: tekton.dev/v1alpha1
  kind: PipelineResource
  metadata:
    name: input
  spec:
    type: url
    params:
      name: resource0
      value: http://example.com/archive.tar.gz
kind: List
---
apiVersion: tekton.dev/v1alpha1
kind: Pipeline
metadata:
  name: pl_txt-pipeline
spec:
  resources:
  - name: input
    type: url
  tasks:
    name: pl_txt
    taskRef:
      name: pl_txt-task
      kind: ""
    resources:
      inputs:
      - name: input
        resource: url
---
apiVersion: tekton.dev/v1alpha1
kind: Pipeline
metadata:
  name: pl_txt-task
spec:
  inputs:
    resources:
    - name: input
      type: url
  outputs: {}
  steps:
  - name: targz
    image: ubuntu
    env:
    - name: foo
      value: bar
    command:
    - command: '["/bin/bash"]'
      args:
      - -c
      - tar xzvf http://example.com/archive.tar.gz
    - command: '["/bin/bash"]'
      args:
      - -c
      - cat source/file.txt
    volumeMounts:
    - name: containers
      mountPath: /var/lib/containers
    arg:
    - name: input
      value: http://example.com/archive.tar.gz
  volumes:
  - name: containers
    hostPath:
      path: containers
      type: unknown
---
apiVersion: tekton.dev/v1alpha1
kind: PipelineRun
metadata:
  name: pl_txt-pipeline-run
spec:
  serviceAccount: kubernetes-user
  timeout: 1h0m0s
  pipelineRef:
    name: pl_txt-pipeline
  trigger:
    type: manual
  resources:
  - name: input
    resourceref:
      name: input
---
```


# The Translation Specification

### ARG <key=value>

Alias for ARGIN
Defines an input argument to the pipeline step as key-value pair.
An entry for the key is made to the PipelineResource, and referred by PipelineRun and Pipeline.
The key and values are passed as arg field to the current pipeline step.

### Example:

Input:

`ARG input=http://example.com/archive.tar.gz`

Output:
In PipelineResource:
```yaml
- apiVersion: tekton.dev/v1alpha1
  kind: PipelineResource
  metadata:
    name: input
  spec:
    type: url
    params:
      name: resource0
      value: http://example.com/archive.tar.gz
```
In Pipeline step:
```yaml
    arg:
    - name: input
      value: http://example.com/archive.tar.gz
```

Again in Pipeline step, after $ variable translation:
```yaml
    - command: '["/bin/bash"]'
      args:
      - -c
      - tar xzvf http://example.com/archive.tar.gz
```

### ARGIN <key=value>

Same as ARG <key=value> .

### ARGOUT <key=value>
Defines an output argument to the pipeline step as key-value pair.
An entry for the key is made to the PipelineResource, and referred by PipelineRun and Pipeline.
The key and values are passed as `arg` field to the current pipeline step.

### ENV <key-value>

Defines an environment variable for the container in the pipeline step.
The key and values are passed as `env` field to the current pipeline step.

### Example:

Input:
`ENV foo=bar`

Output:
In Pipeline step:
```yaml
    env:
    - name: foo
      value: bar
```

### FROM <image>

Defines the container to spin up for the current pipeline step.
The image name is passed as `image` field to the current pipeline step.

### Example:
Input:
FROM ubuntu

Output:
In Pipeline step:
```yaml
  steps:
  - name: targz
    image: ubuntu
```


### LABEL <name>

Defines the name of the current pipeline step. It is a mandate that each step starts with a LABEL
The label name is passed as the `name` field to the current pipeline step.

### Example:
Input:
`LABEL targz`

Output:
In Pipeline step:
```yaml
  steps:
  - name: targz
    image: ubuntu
```

### MOUNT <host=container>

Defines the mount bindings between the host and the container in the current pipeline step.
An entry for `volumeMounts` in the current pipeline step is created with `_host_` as the name and `container` as the `mountPath`
An entry for the `volumes` in the pipeline is created with `_host_` as the name and `host` as the `hostPath`

### Example:
Input:
`MOUNT containers=/var/lib/containers`
Output:
In Pipeline step:

```yaml
    volumeMounts:
    - name: containers
      mountPath: /var/lib/containers
```

Again in the Pipeline step:
```yaml
  volumes:
  - name: containers
    hostPath:
      path: containers
      type: unknown
```

### RUN <commands>

Defines the shell command(s) that will be run in the target container.
A /bin/bash is spawned in the target container, and the entire command string is passed to it.
$ variable translations occur before the command is dispatched, by looking up in the resources.

### Example:
Input:
`RUN tar xzvf $input`
Output:
In the Pipeline step:
```yaml
    command:
    - command: '["/bin/bash"]'
      args:
      - -c
      - tar xzvf http://example.com/archive.tar.gz
```


### USER <user>

Defines a kubernetes cluster service account that will `own` the generated pipeline.
A Role is created with the user.
A RoleBinding is created with the user, that is bound to `cluster-admin` role.

### Example:
Input:
`USER kubernetes-user`

Output:
In Role definition:
```yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: kubernetes-user
```

In RoleBinding:
```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: admin
subjects:
  kind: ServiceAccount
  name: kubernetes-user
  namespace: default
roleRef:
  kind: ClusterRole
  name: cluster-admin
  apiGroup: rbac.authorization.k8s.io
```
