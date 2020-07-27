# Tekton-Hub/frontend
Tekton-Hub is a web based platform for developers to discover, share and contribute tasks and pipelines for Tekton. Tekton is an open-source project for providing a set of shared and standard components for building Kubernetes-style CI/CD systems.

## Key features
* Display Task and Pipelines in a curated way
* Filter tasks on basis of tags
* Search a task on basis of name
* Sort tasks on name, rating and downloads
* Rate a task
* Upload a task

Backend service can be found on [here](https://github.com/tektoncd/experimental/tree/master/hub/api).


## Run locally
* Fork and clone the application in local:
```
  git clone https://github.com/tektoncd/experimental
```

* Go into the project folder and type the following command and further install the npm packages

```
  cd hub/ui/
  npm install
```

* Create a .env file and use the backend route as an environment variable  with variable name REACT_APP_BACKEND_API
* Run the application with following command
```
npm start
```

* Application will be exposed on port 8080.

