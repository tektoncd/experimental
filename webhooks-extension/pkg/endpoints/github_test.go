/*
Copyright 2019 The Tekton Authors
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

package endpoints

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	fakerestclient "k8s.io/client-go/rest/fake"
)

func Test_isGitHubEnterprise(t *testing.T) {
	tests := []struct {
		name   string
		rawurl string
		want   bool
	}{
		{
			rawurl: "https://github.com/owner/repo",
			want:   false,
		},
		{
			rawurl: "https://github.com/owner/repo.git",
			want:   false,
		},
		{
			rawurl: "https://github.company.com/owner/repo",
			want:   true,
		},
		{
			rawurl: "https://github.company.com/owner/repo.git",
			want:   true,
		},
		{
			rawurl: "https://my.company.com/owner/repo",
			want:   true,
		},
		{
			rawurl: "https://hostname/owner/repo",
			want:   true,
		},
		{
			rawurl: "http://hostname/owner/repo",
			want:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.rawurl)
			if err != nil {
				t.Errorf("isGitHubEnterprise() error parsing rawurl %s: %s", tt.rawurl, err)
			}
			if got := isGitHubEnterprise(u); got != tt.want {
				t.Errorf("isGitHubEnterprise() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_getGitHubHubbubAPI(t *testing.T) {
	type args struct {
		rawurl string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			args: args{
				rawurl: "https://github.com/owner/repo",
			},
			want: "https://api.github.com/hub",
		},
		{
			args: args{
				rawurl: "https://github.com/owner/repo.git",
			},
			want: "https://api.github.com/hub",
		},
		{
			args: args{
				rawurl: "https://github.company.com/owner/repo",
			},
			want: "https://github.company.com/api/v3/hub",
		},
		{
			args: args{
				rawurl: "https://github.company.com/owner/repo.git",
			},
			want: "https://github.company.com/api/v3/hub",
		},
		{
			args: args{
				rawurl: "https://my.company.xyz/owner/repo",
			},
			want: "https://my.company.xyz/api/v3/hub",
		},
		{
			args: args{
				rawurl: "https://hostname/owner/repo",
			},
			want: "https://hostname/api/v3/hub",
		},
		{
			args: args{
				rawurl: "http://hostname/owner/repo",
			},
			want: "http://hostname/api/v3/hub",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u, err := url.Parse(tt.args.rawurl)
			if err != nil {
				t.Errorf("getGitHubHubbubAPI() error parsing rawurl %s: %s", tt.args.rawurl, err)
			}
			if got := getGitHubHubbubAPI(u); got != tt.want {
				t.Errorf("getGitHubHubbubAPI() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_doGitHubHubbubRequest(t *testing.T) {
	type hubParams = struct {
		mode     string
		callback string
		secret   string
		events   []string
	}
	tests := []struct {
		name    string
		repoURL string
		params  hubParams
		wantAPI string
	}{
		{
			name:    "subscribe to no events",
			repoURL: "https://github.com/owner/repo",
			params: hubParams{
				mode:     "subscribe",
				callback: "https://examplecallback.com",
				secret:   "mySecret",
				events:   []string{},
			},
			wantAPI: "https://api.github.com/hub",
		},
		{
			name:    "subscribe to one event",
			repoURL: "https://github.com/owner/repo",
			params: hubParams{
				mode:     "subscribe",
				callback: "https://examplecallback.com",
				secret:   "mySecret",
				events:   []string{"push"},
			},
			wantAPI: "https://api.github.com/hub",
		},
		{
			name:    "subscribe public push and pull_request",
			repoURL: "https://github.com/owner/repo",
			params: hubParams{
				mode:     "subscribe",
				callback: "https://examplecallback.com",
				secret:   "mySecret",
				events:   []string{"push", "pull_request"},
			},
			wantAPI: "https://api.github.com/hub",
		},
		{
			name:    "subscribe ghe push and pull_request",
			repoURL: "https://my.company.com/owner/repo",
			params: hubParams{
				mode:     "subscribe",
				callback: "https://examplecallback.com",
				secret:   "mySecret",
				events:   []string{"push", "pull_request"},
			},
			wantAPI: "https://my.company.com/api/v3/hub",
		},
		{
			name:    "unsubscribe ghe push and pull_request",
			repoURL: "https://my.company.com/owner/repo",
			params: hubParams{
				mode:     "unsubscribe",
				callback: "https://examplecallback.com",
				secret:   "mySecret",
				events:   []string{"push", "pull_request"},
			},
			wantAPI: "https://my.company.com/api/v3/hub",
		},
		{
			name:    "unsubscribe ghe push and pull_request",
			repoURL: "https://my.company.com/owner/repo",
			params: hubParams{
				mode:     "unsubscribe",
				callback: "https://examplecallback.com",
				secret:   "mySecret",
				events:   []string{"push", "pull_request"},
			},
			wantAPI: "https://my.company.com/api/v3/hub",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create client
			fakeGitHubClient := fakerestclient.CreateHTTPClient(func(request *http.Request) (*http.Response, error) {
				// Check API URL
				if gotAPI := request.URL.String(); gotAPI != tt.wantAPI {
					t.Errorf("doGitHubHubbubRequest() expected API URL %s; got: %s", tt.wantAPI, gotAPI)
				}
				// Check mode
				if gotMode := request.FormValue("hub.mode"); gotMode != tt.params.mode {
					t.Errorf("doGitHubHubbubRequest() expected hub.mode %s; got: %s", tt.params.mode, gotMode)
				}
				// Check callback
				if gotCallback := request.FormValue("hub.callback"); gotCallback != tt.params.callback {
					t.Errorf("doGitHubHubbubRequest() expected hub.callback %s; got: %s", tt.params.callback, gotCallback)
				}
				// Check secret
				if gotSecret := request.FormValue("hub.secret"); gotSecret != tt.params.secret {
					t.Errorf("doGitHubHubbubRequest() expected hub.secret %s; got: %s", tt.params.secret, gotSecret)
				}
				// Check topic (event & repoURL)
				gotTopic := request.FormValue("hub.topic")
				correctTopic := false
				for _, event := range tt.params.events {
					wantTopic := fmt.Sprintf("%s/events/%s", tt.repoURL, event)
					if gotTopic == wantTopic {
						correctTopic = true
					}
				}
				if !correctTopic {
					t.Errorf("doGitHubHubbubRequest() expected topic with format %s/events/<event>; got: %s", tt.repoURL, gotTopic)
				}
				return &http.Response{
					StatusCode: http.StatusNoContent,
				}, nil
			})
			err := doGitHubHubbubRequest(fakeGitHubClient, tt.repoURL, tt.params.mode, tt.params.callback, tt.params.secret, tt.params.events)
			if err != nil {
				t.Errorf("doGitHubHubbubRequest() returned an error: %s", err)
			}
		})
	}
}

func Test_doGitHubHubbubRequest_error(t *testing.T) {
	// doGitHubHubbubRequest should return an error when the status of the response is not 204
	testStatusCode := func(t *testing.T, statusCode int) {
		t.Run(fmt.Sprintf("statusCode: %d", statusCode), func(t *testing.T) {
			fakeGitHubClient := fakerestclient.CreateHTTPClient(func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: statusCode,
					Status:     http.StatusText(statusCode),
				}, nil
			})
			repoURL := "https://my.company.com/owner/repo"
			mode := "unsubscribe"
			callback := "https://examplecallback.com"
			secret := "mySecret"
			events := []string{"push", "pull_request"}
			err := doGitHubHubbubRequest(fakeGitHubClient, repoURL, mode, callback, secret, events)
			if err == nil {
				t.Errorf("doGitHubHubbubRequest() did not return an error when expected for statusCode %d: %s", statusCode, err)
			}
		})
	}
	for i := 200; i < 209; i++ {
		if i != 204 {
			testStatusCode(t, i)
		}
	}
	for i := 300; i < 309; i++ {
		testStatusCode(t, i)
	}
	for i := 400; i < 452; i++ {
		testStatusCode(t, i)
	}
	for i := 500; i < 512; i++ {
		testStatusCode(t, i)
	}
}
