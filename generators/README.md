# Tekton Generators

This project contains experimental code to create a tool for generating Tekton spec from simplified configs. The goal is to help users bootstrap pipelines in a configurable way.

See [tektoncd/pipeline/#2590](https://github.com/tektoncd/pipeline/issues/2590) information and background.

## Features

This experimental project has been broken down into the features as follows:

1. Parse the yaml file with io.Reader and store the result in the self-defined struct
2. Create tool that given an input spec with steps, generates the resulting Tekton resources for the particular type.