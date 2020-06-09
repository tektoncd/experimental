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
	"testing"

	"github.com/jenkins-x/go-scm/scm"
	"github.com/jenkins-x/go-scm/scm/driver/github"
	"github.com/jenkins-x/go-scm/scm/driver/gitlab"
)

func TestCreateClient(t *testing.T) {
	tests := []struct {
		desc     string
		repoType string
		want     *scm.Client
	}{
		{"github repository", "github", github.NewDefault()},
		{"gitlab repository", "gitlab", gitlab.NewDefault()},
		{"unsupported repository", "abcd", nil},
	}
	for i, tt := range tests {
		t.Run(fmt.Sprintf("Test %d", i), func(rt *testing.T) {
			got := createClient("123", tt.repoType)
			if tt.want == nil || got == nil {
				if tt.want != got {
					rt.Fatalf("expected no client but got %v", got)
				}
			} else {
				if tt.want.Driver.String() != got.Driver.String() {
					rt.Fatalf("createClient() failed: got client %v, want %v", tt.want.Driver.String(), got.Driver.String())
				}
			}
		})
	}

}
