// Copyright 2020 The Tekton Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package pipelinerun

import (
	"fmt"
	"net/url"
	"strings"

	pipelinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1alpha1"
)

// findGitResource locates a Git PipelineResource in a PipelineRun.
//
// If no Git resources are found, an error is returned.
// If more than one Git resource is found, an error is returned.
func findGitResource(p *pipelinev1.PipelineRun) (*pipelinev1.PipelineResourceSpec, error) {
	var spec *pipelinev1.PipelineResourceSpec
	for _, r := range p.Spec.Resources {
		if r.ResourceSpec == nil {
			continue
		}
		if r.ResourceSpec.Type == pipelinev1.PipelineResourceTypeGit {
			if spec != nil {
				return nil, fmt.Errorf("found multiple git PipelineResources in the PipelineRun %s", p.ObjectMeta.Name)
			}
			spec = r.ResourceSpec
		}
	}
	if spec == nil {
		return nil, fmt.Errorf("failed to find a git PipelineResource in the PipelineRun %s", p.ObjectMeta.Name)
	}

	return spec, nil
}

// TODO This only parses GitHub repo paths, would need work to parse GitLab repo
// paths too (can have more components).
func getRepoAndSHA(p *pipelinev1.PipelineResourceSpec) (string, string, error) {
	if p.Type != pipelinev1.PipelineResourceTypeGit {
		return "", "", fmt.Errorf("failed to get repo and SHA from non-git resource: %s", p)
	}
	u, err := getResourceParamByName(p.Params, "url")
	if err != nil {
		return "", "", fmt.Errorf("failed to find param url in getRepoAndSHA: %w", err)
	}

	rev, err := getResourceParamByName(p.Params, "revision")
	if err != nil {
		return "", "", fmt.Errorf("failed to find param revision in getRepoAndSHA: %w", err)
	}
	return strings.TrimSuffix(u, ".git"), rev, nil
}

func getResourceParamByName(params []pipelinev1.ResourceParam, name string) (string, error) {
	for _, p := range params {
		if p.Name == name {
			return p.Value, nil
		}
	}
	return "", fmt.Errorf("no resource parameter with name %s", name)
}

func extractRepoPath(s string) (string, error) {
	u, err := url.Parse(s)
	if err != nil {
		return "", fmt.Errorf("failed to parse repo URL %s: %w", s, err)
	}
	parts := strings.Split(u.Path, "/")
	components := []string{}
	for i := 1; i < len(parts); i++ {
		if parts[i] != "" {
			components = append(components, parts[i])
		}
	}
	if len(components) < 2 {
		return "", fmt.Errorf("could not determine repo from URL: %v", u)
	}
	return strings.Join(components, "/"), nil
}

func getDriverName(rawURL string) (string, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", err
	}
	if s := strings.TrimSuffix(u.Host, ".com"); s != u.Host {
		return strings.ToLower(s), nil
	}
	return "", fmt.Errorf("unable to determine type of Git host from: %s", rawURL)
}
