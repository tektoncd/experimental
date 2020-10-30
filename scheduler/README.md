# coscheduler-same-node
The repo is for trying to implements such a scheduler:
https://github.com/tektoncd/pipeline/issues/3052

# Installation
```
ko apply -f config/
```

# Take a try
`kubectl create -f examples/pods.yaml`

# Description
The `scheduler` is based on [Scheduler plugin framework](https://github.com/kubernetes/enhancements/blob/master/keps/sig-scheduling/20180409-scheduling-framework.md)

Use special labels of `pod` to mark the group of pods.
```
labels:
     pod-group.scheduling.sigs.k8s.io/name: test
     pod-group.scheduling.sigs.k8s.io/total: "2"
```

The `pod`s in same `pod-group` will be scheduler to same node if the node can satisfy the resource requirement of whole group, or all pods in the group will `pending`.
