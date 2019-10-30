// Copyright © 2019 IBM Corporation and others.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// Master structure that represents the entire pipeline
type PlGen struct {
	pr          PipelineResource
	pspecs      Spec
	psteps      []Steps
	plt         PipelineTask
	pl          Pipeline
	role        Role
	rolebinding RoleBinding
	plr         PipelineRun
}

// Used by PipelineResource,
type Metadata struct {
	Name string `yaml:"name"`
}

// Used by PipelineResource, through PipelineResource->Items->PipelineResourceSpec
type PipelineResourceParams struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// Used by PipelineResource, through PipelineResource->Items
type PipelineResourceSpec struct {
	Type   string                   `yaml:"type"`
	Params []PipelineResourceParams `yaml:"params"`
}

// Used by PipelineResource, top level to PipelineResource
type Items struct {
	APIVersion string               `yaml:"apiVersion"`
	Kind       string               `yaml:"kind"`
	Metadata   Metadata             `yaml:"metadata"`
	Spec       PipelineResourceSpec `yaml:"spec"`
}

// Used by PlGen, top level.
type PipelineResource struct {
	APIVersion string  `yaml:"apiVersion"`
	Items      []Items `yaml:"items"`
	Kind       string  `yaml:"kind"`
}

// Used by PlGen, top level
type Role struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Metadata   Metadata `yaml:"metadata"`
}

// Used by RoleBinding, through Rolebinding
type Subjects struct {
	Kind      string `yaml:"kind"`
	Name      string `yaml:"name"`
	Namespace string `yaml:"namespace"`
}

// Used by RoleBinding, through Rolebinding
type RoleRef struct {
	Kind     string `yaml:"kind"`
	Name     string `yaml:"name"`
	APIGroup string `yaml:"apiGroup"`
}

// Used by PlGen, top level
type RoleBinding struct {
	APIVersion string     `yaml:"apiVersion"`
	Kind       string     `yaml:"kind,omitempty"`
	Metadata   Metadata   `yaml:"metadata,omitempty"`
	Subjects   []Subjects `yaml:"subjects,omitempty"`
	RoleRef    RoleRef    `yaml:"roleRef,omitempty"`
}

// Used by Pipeline, through Pipeline->PipelineSpec->Tasks
type Params struct {
	Name  string `yaml:"name"`
	Value string `yaml:"default"`
}

// Used by Pipeline, through Pipeline->PipelineSpec->Tasks->PipelineResources
type PipelineInputs struct {
	Name     string `yaml:"name"`
	Resource string `yaml:"resource,omitempty"`
}

// Used by Pipeline, through Pipeline->PipelineSpec->Tasks->PipelineResources
type PipelineOutputs struct {
	Name     string `yaml:"name"`
	Resource string `yaml:"resource,omitempty"`
}

// Used by Pipeline, through Pipeline->PipelineSpec->Tasks
type PipelineResources struct {
	Inputs  []PipelineInputs  `yaml:"inputs,omitempty"`
	Outputs []PipelineOutputs `yaml:"outputs,omitempty"`
}

// Used by Pipeline, through Pipeline->PipelineSpec->Tasks
type TaskRef struct {
	Name string `yaml:"name"`
	Kind string `yaml:"kind"`
}

// Used by Pipeline, through Pipeline->PipelineSpec
type Resources struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
}

// Used by Pipeline, through Pipeline->PipelineSpec
type Tasks struct {
	Name      string            `yaml:"name,omitempty"`
	Taskref   TaskRef           `yaml:"taskRef,omitempty"`
	Resources PipelineResources `yaml:"resources,omitempty"`
	Params    []Params          `yaml:"params,omitempty"`
}

// Used by Pipeline, through Pipeline
type PipelineSpec struct {
	Resources []Resources `yaml:"resources,omitempty"`
	Tasks     [1]Tasks    `yaml:"tasks,omitempty"`
}

// Used by PlGen, top level
type Pipeline struct {
	APIVersion string       `yaml:"apiVersion"`
	Kind       string       `yaml:"kind"`
	Meta       Metadata     `yaml:"metadata"`
	Spec       PipelineSpec `yaml:"spec"`
}

// Used by PipelineTask, through PipelineTask->Spec->Steps
type Arg struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// Used by PipelineTask, through PipelineTask->Spec->Steps
type Mount struct {
	Name  string `yaml:"name"`
	Value string `yaml:"mountPath"`
}

// Used by PipelineTask, through PipelineTask->Spec->Steps
type Env struct {
	Name  string `yaml:"name"`
	Value string `yaml:"value"`
}

// Used by PipelineTask, through PipelineTask->Spec->Volumes
type HostPath struct {
	Path string `yaml:"path"`
	Type string `yaml:"type"`
}

// Used by PipelineTask, through PipelineTask->Spec
type Volumes struct {
	Name     string   `yaml:"name"`
	HostPath HostPath `yaml:"hostPath"`
}

// Used by PipelineTask, through PipelineTask->Spec
type Steps struct {
	Name    string   `yaml:"name"`
	Image   string   `yaml:"image"`
	Env     []Env    `yaml:"env,omitempty"`
	Command []string `yaml:"command,omitempty`
	Args    []string `yaml:"args,omitempty"`
	Mount   []Mount  `yaml:"volumeMounts,omitempty"`
	Arg     []Arg    `yaml:"arg,omitempty"`
}

// Used by PipelineTask, through PipelineTask->Spec
type Inputs struct {
	Resources []Resources `yaml:"resources,omitempty"`
	Params    []Params    `yaml:"params,omitempty"`
}

// Used by PipelineTask, through PipelineTask->Spec
type Outputs struct {
	Resources []Resources `yaml:"resources,omitempty"`
	Params    []Params    `yaml:"params,omitempty"`
}

// Used by PipelineTask, through PipelineTask
type Spec struct {
	Inputs  Inputs    `yaml:"inputs,omitempty"`
	Outputs Outputs   `yaml:"outputs,omitempty"`
	Steps   []Steps   `yaml:"steps,omitempty"`
	Volumes []Volumes `yaml:"volumes,omitempty"`
}

// Used by PlGen, top level
type PipelineTask struct {
	APIVersion string   `yaml:"apiVersion"`
	Kind       string   `yaml:"kind"`
	Meta       Metadata `yaml:"metadata"`
	Spec       Spec     `yaml:"spec"`
}

// Used by PipelineRun, through PipelineRun->PipelineRunSpec->PipelineRunResources
type PipelineRunResourceRef struct {
	Name string `yaml:"name"`
}

// Used by PipelineRun, through PipelineRun->PipelineRunSpec
type PipelineRunResources struct {
	Name        string                 `yaml:"name"`
	ResourceRef PipelineRunResourceRef `string:"resourceRef"`
}

// Used by PipelineRun, through PipelineRun->PipelineRunSpec
type PipelineRunPipelineRef struct {
	Name string `yaml:"name"`
}

// Used by PipelineRun, through PipelineRun
type PipelineRunSpec struct {
	ServiceAccount string                 `yaml:"serviceAccount"`
	Timeout        string                 `yaml:"timeout"`
	PipelineRef    PipelineRunPipelineRef `yaml:"pipelineRef"`
	Resources      []PipelineRunResources `yaml:"resources,omitempty"`
}

// Used by PlGen, top level
type PipelineRun struct {
	APIVersion string          `yaml:"apiVersion"`
	Kind       string          `yaml:"kind"`
	Metadata   Metadata        `yaml:"metadata"`
	Spec       PipelineRunSpec `yaml:"spec"`
}

// Marshals a GO struct into YAML definition
// and prints a --- separator. Any error, exit
func Marshal(in interface{}) {
	data, err := yaml.Marshal(in)
	if err != nil {
		fmt.Println("Error while marshalling into yaml:", err)
		os.Exit(1)
	}
	fmt.Print(string(data))
	fmt.Println("---")
}

// Generates a Role from a USER verb
func GenRole(plg PlGen) {
	role := plg.role
	Marshal(&role)
}

// Generates a RoleBinding from a USER verb
func GenRoleBinding(plg PlGen) {
	rolebinding := plg.rolebinding
	Marshal(&rolebinding)
}

// Generates a pipeline, binds it with a pipeline task
func GenPipeline(plg PlGen) {
	pl := plg.pl
	pl.APIVersion = apiVersion
	pl.Kind = "Pipeline"
	pl.Meta.Name = nomenClature + "-pipeline"
	pl.Spec.Tasks[0].Name = nomenClature
	pl.Spec.Tasks[0].Taskref.Name = nomenClature + "-task"
	pl.Spec.Tasks[0].Taskref.Kind = "Task"
	Marshal(&pl)
}

// Generates a pipeline run, binds it with a pipeline
func GenPipelineRun(plg PlGen) {
	plr := plg.plr
	plr.APIVersion = apiVersion
	plr.Kind = "PipelineRun"
	plr.Metadata.Name = nomenClature + "-pipeline-run"
	plr.Spec.Timeout = pipelineTimeout
	plr.Spec.PipelineRef.Name = nomenClature + "-pipeline"
	Marshal(&plr)
}

// Arg was used for internal purposes:
// spreading the input/output,
// as a cache for dollar variable lookup
// remove this before pipeline generation
// use normal for loop as opposed to iterator
// to effect the change in the actual structure
func removeArgs(plg *PlGen) {
	steps := plg.pspecs.Steps
	for i := 0; i < len(steps); i++ {
		steps[i].Arg = []Arg{}
	}
}

// Generates a pipeline task
func GenPipelineTask(plg PlGen) {
	plg.pspecs.Steps = plg.psteps
	plg.plt.Spec = plg.pspecs
	removeArgs(&plg)
	plt := plg.plt
	plt.APIVersion = apiVersion
	plt.Kind = "Task"
	plt.Meta.Name = nomenClature + "-task"
	Marshal(&plt)
}

// Generates a list of pipeline resources
func GenResource(plg PlGen) {
	pr := plg.pr
	pr.APIVersion = "v1"
	pr.Kind = "List"
	Marshal(&pr)
}

// Transform a RUN step. Basically:
// 1. Transalate any $ variables
// search in the old steps, and the current one
// 2. suffix the commands under /bin/bash
func TransformRun(line string, step *Steps, steps *[]Steps) {
	u := strings.Split(line, " ")[1:]
	var v = strings.Join(u, " ")
	for _, old := range u {
		re := regexp.MustCompile("\\$([^\\s]+)")
		words := re.FindAllString(old, -1)
		for _, token := range words {
			v = strings.ReplaceAll(v, token, replace(steps, token))
			current := []Steps{
				*step,
			}
			v = strings.ReplaceAll(v, token, replace(&current, token))
		}
	}
	step.Command = []string{"/bin/bash"}
	step.Args = append(step.Args, "-c", v)
	debuglog("processing RUN", v, "as", step.Command)
}

// Transform a USER verb.
// Create a Cluster Role
// Bind it with cluster-admin privilege
func TransformRole(line string, plg *PlGen) {
	name := strings.Split(line, " ")[1]
	var role Role
	var rolebinding RoleBinding
	role.APIVersion = "v1"
	role.Kind = "ServiceAccount"
	role.Metadata = Metadata{Name: name}

	var sub Subjects
	sub.Kind = role.Kind
	sub.Name = name
	sub.Namespace = namespace

	var roleref RoleRef
	roleref.Kind = "ClusterRole"
	roleref.Name = "cluster-admin"
	roleref.APIGroup = "rbac.authorization.k8s.io"

	rolebinding.APIVersion = "rbac.authorization.k8s.io/v1"
	rolebinding.Kind = "ClusterRoleBinding"
	rolebinding.Metadata.Name = rolebindingname
	rolebinding.Subjects = []Subjects{sub}
	rolebinding.RoleRef = roleref

	plg.plr.Spec.ServiceAccount = role.Metadata.Name
	plg.role = role
	plg.rolebinding = rolebinding

	debuglog("processing USER", name, "as ClusterRoleBinding")
}

// Transform a MOUNT verb.
// For a MOUNT A=B:
// Create a volume with the name as _A_ and hostMount as A
// Create a voumeMount with name as _A_ and mountPath as B
func TransformMount(step *Steps, name string, val string, plg *PlGen) {
	mname := strings.ReplaceAll(name, "/", "_")
	step.Mount = append(step.Mount, Mount{Name: mname, Value: val})
	volumes := Volumes{Name: mname, HostPath: HostPath{Path: name, Type: "unknown"}}
	plg.pspecs.Volumes = append(plg.pspecs.Volumes, volumes)
	debuglog("processing MOUNT", mname, "as", volumes, "and", step.Mount)
}

// Transform an ENV verb
// Also record the key values in ARG verb, for future $ translations
func TransformEnv(step *Steps, name string, val string, plg *PlGen) {
	step.Env = append(step.Env, Env{Name: name, Value: val})
	step.Arg = append(step.Arg, Arg{Name: name, Value: val})
	debuglog("processing ENV", step.Env)
}

// Transform ARG, ARGIN, ARGOUT verbs
// 1. From the value, try to `decipher` its resource type:
// ref: https://github.com/tektoncd/pipeline/blob/master/docs/resources.md
// 2. compose headers
// 3. create an artifical name for the resource
// 4. bind the resource as input or output appropriately
// 5. later, this will be looked up for $ variable translations.
func TransformArg(step *Steps, name string, key string, val string, plg *PlGen) {
	itemName := "resource" + strconv.Itoa(rindex)
	rindex++
	itemType := guessItemType(val)
	params := []PipelineResourceParams{{Name: itemName, Value: val}}
	plg.pr.Items = append(plg.pr.Items, Items{
		APIVersion: "tekton.dev/v1alpha1",
		Kind:       "PipelineResource",
		Metadata:   Metadata{Name: name},
		Spec:       PipelineResourceSpec{Params: params, Type: itemType}})
	plrrr := PipelineRunResourceRef{Name: name}
	plrr := PipelineRunResources{Name: name, ResourceRef: plrrr}
	plg.plr.Spec.Resources = append(plg.plr.Spec.Resources, plrr)
	r := Resources{Name: name, Type: itemType}
	plg.pl.Spec.Resources = append(plg.pl.Spec.Resources, r)
	if key == "ARGIN" || key == "ARG" {
		r := Resources{Name: name, Type: itemType}
		t := append(plg.pspecs.Inputs.Resources, r)
		plg.pspecs.Inputs.Resources = t
		pi := PipelineInputs{Name: name, Resource: name}
		pq := append(plg.pl.Spec.Tasks[0].Resources.Inputs, pi)
		plg.pl.Spec.Tasks[0].Resources.Inputs = pq
	} else {
		r := Resources{Name: name, Type: itemType}
		s := append(plg.pspecs.Outputs.Resources, r)
		plg.pspecs.Outputs.Resources = s
		plo := PipelineOutputs{Name: name, Resource: name}
		plp := append(plg.pl.Spec.Tasks[0].Resources.Outputs, plo)
		plg.pl.Spec.Tasks[0].Resources.Outputs = plp
	}
	step.Arg = append(step.Arg, Arg{Name: name, Value: val})
	debuglog("processing ARG", step.Arg)
}

func guessItemType(item string) string {
	if strings.Contains(item, "github.com") {
		if strings.Contains(item, "/pull/") {
			return "pullRequest"
		}
		return "git"
	} else if strings.Contains(item, "docker.io") ||
		strings.Contains(item, "gcr.io") ||
		strings.Contains(item, "registry.access.redhat.com") {
		return "image"
	}
	// TODO: detect storage, cloud event, cluster patterns.
	return "unknown"
}

// Main translation loop. As we are not mandating any specific order
// in the pipeline script, the top level PlGen object is pre-created
// and passed to this so that as and when data elements (more import-
// antly resources) are encountered, they can be attached to it.
func transformSteps(plg *PlGen, stepstr string, index int) {
	var step Steps
	lines := strings.Split(stepstr, "\n")
	for _, line := range lines {
		if line != "" {
			key := strings.Split(line, " ")[0]
			switch key {
			case "LABEL":
				step.Name = strings.Split(line, " ")[1]
				debuglog("processing LABEL", step.Name)
			case "FROM":
				step.Image = strings.Split(line, " ")[1]
				debuglog("processing FROM", step.Image)
			case "ARG", "ARGIN", "ARGOUT", "ENV", "MOUNT":
				value := strings.Split(line, " ")[1]
				name := strings.Split(value, "=")[0]
				val := strings.Split(value, "=")[1]
				if strings.HasPrefix(key, "ARG") {
					TransformArg(&step, name, key, val, plg)
				} else if key == "ENV" {
					TransformEnv(&step, name, val, plg)
				} else {
					TransformMount(&step, name, val, plg)
				}
			case "RUN":
				TransformRun(line, &step, &plg.psteps)
			case "USER":
				TransformRole(line, plg)
			default:
				fmt.Println("bad pipeline verb:", key)
				os.Exit(1)
			}
		}
	}
	plg.psteps = append(plg.psteps, step)
}
