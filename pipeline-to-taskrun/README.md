# Pipeline to TaskRun

This project will be a controller that enables an experimental [custom task](https://github.com/tektoncd/pipeline/blob/main/docs/runs.md)
that will allow you to execute a Pipeline (with [limited features](#supported-pipeline-features)) via a TaskRun, enabling you to
run a Pipeline in a pod ([TEP-0044](https://github.com/tektoncd/community/blob/main/teps/0044-decouple-task-composition-from-scheduling.md)).
