/*
Copyright 2021 The Tekton Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package gitref

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/go-git/go-billy/v5/memfs"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/tektoncd/experimental/remote-resolution/pkg/reconciler/framework"
)

type Resolver struct{}

func (r *Resolver) Initialize(ctx context.Context) error {
	return nil
}

func (r *Resolver) GetName() string {
	return "Git"
}

func (r *Resolver) GetSelector() map[string]string {
	return map[string]string{
		"resolution.tekton.dev/type": "git",
	}
}

func (r *Resolver) ValidateParams(params map[string]string) error {
	required := []string{
		"repo",
		"path",
	}
	missing := []string{}
	if params == nil {
		missing = required
	} else {
		for _, p := range required {
			v, has := params[p]
			if !has || v == "" {
				missing = append(missing, p)
			}
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("missing %v", strings.Join(missing, ", "))
	}

	if params["commit"] != "" && params["branch"] != "" {
		return fmt.Errorf("supplied both commit and branch")
	}

	// TODO: validate repo url is well-formed, git:// or https://
	// TODO: validate path is valid relative path

	return nil
}

func (r *Resolver) Resolve(params map[string]string) (string, map[string]string, error) {
	repo := params["repo"]
	commit := params["commit"]
	branch := params["branch"]
	path := params["path"]
	cloneOpts := &git.CloneOptions{
		URL: repo,
	}
	filesystem := memfs.New()
	if branch != "" {
		cloneOpts.SingleBranch = true
		cloneOpts.ReferenceName = plumbing.NewBranchReferenceName(branch)
	}
	repository, err := git.Clone(memory.NewStorage(), filesystem, cloneOpts)
	if err != nil {
		return "", nil, fmt.Errorf("clone error: %w", err)
	}
	if commit == "" {
		headRef, err := repository.Head()
		if err != nil {
			return "", nil, fmt.Errorf("HEAD error: %w", err)
		}
		commit = headRef.Hash().String()
	}

	w, err := repository.Worktree()
	if err != nil {
		return "", nil, fmt.Errorf("worktree error: %v", err)
	}

	err = w.Checkout(&git.CheckoutOptions{
		Hash: plumbing.NewHash(commit),
	})
	if err != nil {
		return "", nil, fmt.Errorf("checkout error: %v", err)
	}

	f, err := filesystem.Open(path)
	if err != nil {
		return "", nil, fmt.Errorf("error opening file %q: %v", path, err)
	}

	sb := &strings.Builder{}
	_, err = io.Copy(sb, f)
	if err != nil {
		return "", nil, fmt.Errorf("error reading file %q: %v", path, err)
	}

	annotations := map[string]string{
		"commit": commit,
	}

	maybeYAML := sb.String()
	j, err := yaml.YAMLToJSON([]byte(maybeYAML))
	if err != nil {
		annotations["content-type"] = "application/x-yaml"
		return maybeYAML, annotations, nil
	}

	annotations["content-type"] = "application/json"
	return string(j), annotations, nil
}

var _ framework.Resolver = &Resolver{}
