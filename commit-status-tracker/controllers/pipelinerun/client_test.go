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
	"github.com/jenkins-x/go-scm/scm/transport"
)

func TestCreateClient(t *testing.T) {
	tests := []struct {
		name       string
		repoURL    string
		baseURL    string
		wantDriver scm.Driver

		wantErr   string
		wantToken string
	}{
		{
			"github repository",
			"https://github.com/test/test.git", "https://api.github.com/", scm.DriverGithub, "", "",
		},
		{
			"gitlab repository", "https://gitlab.com/test/test.git", "https://gitlab.com/", scm.DriverGitlab, "", "token",
		},
		{
			"unsupported repository",
			"https://example.com/test/test.git", "", scm.DriverUnknown, "unable to identify driver from hostname: example.com", "",
		},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s", tt.name), func(rt *testing.T) {
			got, err := createClient(tt.repoURL, "token")
			if tt.wantErr != "" {
				if err.Error() != tt.wantErr {
					rt.Errorf("error did not match, got %q, want %q", err, tt.wantErr)
				}
				return
			}
			if got.BaseURL.String() != tt.baseURL {
				rt.Errorf("BaseURL got %q, want %q", got.BaseURL, tt.baseURL)
			}
			if got.Driver != tt.wantDriver {
				rt.Errorf("Driver got %q, want %q", got.Driver, tt.wantDriver)
			}

			if tt.wantToken != "" {
				if p := got.Client.Transport.(*transport.PrivateToken).Token; p != "token" {
					t.Fatalf("got %q, want %q", p, "token")
				}
			}
		})
	}
}

func TestAddTokenToURL(t *testing.T) {
	testURL := "https://gitlab.com/org/repo"
	token := "test-token"

	newURL, err := addTokenToURL(testURL, token)
	if err != nil {
		t.Fatal(err)
	}

	if newURL != "https://:test-token@gitlab.com/org/repo" {
		t.Fatalf("got %q", newURL)
	}
}
