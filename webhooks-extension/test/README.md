# Testing and scripting

This directory will contain scripts used for several related purposes. 

- As a developer I want to set up a local test environment
  - From clean: install all prerequisites
  - Having prereqs installed, set up a pipeline and webhook for a simple test repository
- We'll want automated tests that do much the same things. 

This is a work in progress and will take a while to settle down. 

## Clean Docker Desktop: Install prereqs

- check test/config.sh 
- `test/install_prereqs.sh`

## Install Dashboard and wehooks extension

- you need a docker hub ID and will be prompted to 'docker login' if you've not all ready done so
- check that GOPATH is set
- `export KO_DOCKER_REPO=docker.io/[your_dockerhub_id]`
- `test/install_dashboard_and_extension.sh`