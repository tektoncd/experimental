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
	"io/ioutil"
	"os"
	"regexp"
	"strings"
)

// Some static assumptions used in the tool
const (
	apiVersion      = "tekton.dev/v1alpha1"
	namespace       = "default"
	pipelineTimeout = "1h0m0s"
	pipelineTrigger = "manual"
	rolebindingname = "admin"
	apiGroup        = "rbac.authorization.k8s.io"
)

// Name of the pipeline, pipelinerun, pipelinetask etc.
// are gleaned from the input file name.
// a future optimization is to replace this with a proper
// flag that receives the name of the pipeline objects
var (
	nomenClature = "test"
	debug        = false
	rindex       = 0
)

// Valid verbs at the moment. When adding one, all
// what you need to make sure is you have an entry
// for it in the switch case of transformSteps.
var verbs = []string{"ARG", "ARGIN", "ARGOUT", "FROM", "RUN", "LABEL", "ENV", "MOUNT", "USER"}

func replace(steps *[]Steps, variable string) string {
	if steps == nil {
		return variable
	}
	for _, step := range *steps {
		for _, e := range step.Arg {
			if "$"+e.Name == variable {
				return e.Value
			}
		}
	}
	return variable
}

func isValidVerb(a string) bool {
	for _, b := range verbs {
		if b == a {
			return true
		}
	}
	return false
}

func validateSanity(step string, sindex int) {

	// sanity #1: verbs start with LABEL
	if !strings.HasPrefix(step, "LABEL ") {
		fmt.Println("Pipeline steps start with LABEL verb")
		os.Exit(1)
	}

	// sanity #2: verbs only from within a known subset
	lines := strings.Split(step, "\n")
	for lindex, line := range lines {
		var verbs = strings.Split(line, " ")
		verb := strings.TrimSpace(verbs[0])
		if len(verb) != 0 && !isValidVerb(verb) {
			fmt.Println("Invalid pipeline verb", verb, "in step #", sindex+1, "line #", lindex+1)
			os.Exit(1)
		}
	}

	// sanity #3: no empty verbs
	for lindex, line := range lines {
		var verbs = strings.Split(line, " ")
		if strings.TrimSpace(verbs[0]) != "" && len(verbs) <= 1 {
			verb := strings.TrimSpace(verbs[0])
			fmt.Println("Verb", verb, "does not have a value in step #", sindex+1, "line #", lindex+1)
			os.Exit(1)
		}
	}
}
func debuglog(args ...interface{}) {
	if debug {
		fmt.Print("[Debug] ")
		fmt.Println(args...)
	}
}

func main() {

	// Our master pipeline db
	var plg PlGen

	// Validate proper usage
	count := len(os.Args)
	if count != 2 && count != 3 {
		fmt.Println("Usage: plgen <filename> [-v]")
		os.Exit(1)
	}

	if count == 3 {
		debug = true
	}

	nomenClature = strings.ReplaceAll(os.Args[1], ".", "-")

	// Get the file data
	filename := os.Args[1]
	rawdata, err := ioutil.ReadFile(filename)
	if err != nil {
		fmt.Println("error reading input file:", err)
		os.Exit(1)
	}
	debuglog("reading", filename)
	data := string(rawdata)

	// Get the data split as steps
	re := regexp.MustCompile("\nLABEL ")
	steps := re.Split(data, -1)
	for index := range steps {
		if index > 0 {
			steps[index] = "LABEL " + steps[index]
		}
	}

	// Do basic sanity on the data
	debuglog("doing basic sanity checking on the input")
	for index, step := range steps {
		validateSanity(step, index)
	}

	// Perform transformation with 'step' as the unit of work
	for index, step := range steps {
		transformSteps(&plg, step, index)
	}

	// Print the pipeline definition
	debuglog("generating the pipeline data")
	debuglog("---")
	GenRole(plg)
	GenRoleBinding(plg)
	GenResource(plg)
	GenPipeline(plg)
	GenPipelineTask(plg)
	GenPipelineRun(plg)
}
