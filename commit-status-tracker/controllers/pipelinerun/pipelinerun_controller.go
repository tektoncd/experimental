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
	"context"
	"crypto/sha1"
	"crypto/tls"
	"fmt"
	"github.com/jenkins-x/go-scm/scm/factory"
	"github.com/spf13/pflag"
	"golang.org/x/oauth2"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	"net/url"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/jenkins-x/go-scm/scm"
	pipelinesv1alpha1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
)

var (
	log = logf.Log.WithName("controller_pipelinerun")

	flagSet *pflag.FlagSet

	insecureTLS bool
)

func init() {
	flagSet = pflag.NewFlagSet("status-tracker", pflag.ExitOnError)
	flagSet.BoolVar(&insecureTLS, "insecure", false, "Disable verification of remote TLS certificates")
}

// FlagSet - The flags for this controller.
func FlagSet() *pflag.FlagSet {
	return flagSet
}

type pipelineRunTracker map[string]State

type scmClientFactory func(repoURL string, authToken string) (*scm.Client, error)

// PipelinerunReconciler reconciles a CommitStatusTracker object
type PipelinerunReconciler struct {
	Client       client.Client
	Scheme       *runtime.Scheme
	scmFactory   scmClientFactory
	pipelineRuns pipelineRunTracker
}

func NewReconciler(mgr manager.Manager) *PipelinerunReconciler {
	return &PipelinerunReconciler{
		Client:       mgr.GetClient(),
		Scheme:       mgr.GetScheme(),
		scmFactory:   createClient,
		pipelineRuns: make(pipelineRunTracker),
	}
}

//+kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns,verbs=get;list;watch
//+kubebuilder:rbac:groups=tekton.dev,resources=pipelineruns/status,verbs=get
//+kubebuilder:rbac:groups=tekton.dev,resources=taskruns,verbs=get;list;watch
//+kubebuilder:rbac:groups=tekton.dev,resources=taskruns/status,verbs=get
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile reads that state of the cluster for a PipelineRun object and makes changes based on the state read
// and what is in the PipelineRun.Spec
func (r *PipelinerunReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = logf.FromContext(ctx)

	// your logic here
	reqLogger := log.WithValues("Request.Namespace", req.Namespace, "Request.Name", req.Name)
	reqLogger.Info("Reconciling PipelineRun")

	// Fetch the PipelineRun instance
	pipelineRun := &pipelinesv1alpha1.PipelineRun{}
	err := r.Client.Get(ctx, req.NamespacedName, pipelineRun)
	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	if !isNotifiablePipelineRun(pipelineRun) {
		reqLogger.Info("not a notifiable pipeline run")
		return ctrl.Result{}, nil
	}
	var repoURL string
	var sha string
	gitRepoAnnotated := isGitRepoConfiguredViaAnnotation(pipelineRun)
	gitRevisionAnnotated := isGitRevisionConfiguredViaAnnotation(pipelineRun)
	if (gitRepoAnnotated && !gitRevisionAnnotated) || (!gitRepoAnnotated && gitRevisionAnnotated) {
		reqLogger.Info("failed to use git repository and git revision from annotations. tekton.dev/git-repo or tekton.dev/git-revision missing")
	}
	if gitRepoAnnotated && gitRevisionAnnotated {
		repoURL = getAnnotationByName(pipelineRun, gitRepoToReportTo, "")
		sha = getAnnotationByName(pipelineRun, gitRevision, "")
	} else {
		res, err := findGitResource(pipelineRun)
		if err != nil {
			reqLogger.Error(err, "failed to find a git resource")
			return reconcile.Result{}, nil
		}
		repoURLFromResource, shaFromResource, err := getRepoAndSHA(res)
		if err != nil {
			reqLogger.Error(err, "failed to parse the URL and SHA correctly")
			return reconcile.Result{}, nil
		}
		repoURL = repoURLFromResource
		sha = shaFromResource
	}
	repo, err := extractRepoPath(repoURL)
	if err != nil {
		reqLogger.Error(err, "failed to extract repository path")
		return ctrl.Result{}, nil
	}
	key := keyForCommit(repo, sha)
	status := getPipelineRunState(pipelineRun)
	last, ok := r.pipelineRuns[key]
	// This uses the in-memory state to retain an original pending state to
	// avoid duplicate API calls.
	if ok {
		if status == last {
			return ctrl.Result{}, nil
		}
	}

	secret, err := getAuthSecret(r.Client, req.Namespace)
	if err != nil {
		reqLogger.Error(err, "failed to get an authSecret")
		return ctrl.Result{}, nil
	}

	gitClient, err := r.scmFactory(repoURL, secret)
	if err != nil {
		reqLogger.Error(err, "failed to create gitClient to send commit-status")
		return ctrl.Result{}, nil
	}
	commitStatusInput := getCommitStatusInput(pipelineRun)
	reqLogger.Info("creating a commit status for", "repo", repo, "sha", sha, "status", commitStatusInput)
	_, _, err = gitClient.Repositories.CreateStatus(ctx, repo, sha, commitStatusInput)
	if err != nil {
		reqLogger.Error(err, "failed to create the commit status")
		return ctrl.Result{}, err
	}
	r.pipelineRuns[key] = status
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *PipelinerunReconciler) SetupWithManager(mgr ctrl.Manager) error {
	c, err := controller.New("pipelinerun-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &pipelinesv1alpha1.PipelineRun{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}
	return nil
}
func keyForCommit(repo, sha string) string {
	return sha1String(fmt.Sprintf("%s:%s", repo, sha))
}

func sha1String(s string) string {
	return fmt.Sprintf("%x", sha1.Sum([]byte(s)))
}

func createClient(repoURL, token string) (*scm.Client, error) {
	newURL, err := addTokenToURL(repoURL, token)
	if err != nil {
		return nil, err
	}
	cli, err := factory.FromRepoURL(newURL)
	if insecureTLS {
		cli.Client = makeInsecureClient(token)
	}
	return cli, err
}

func addTokenToURL(s, token string) (string, error) {
	parsed, err := url.Parse(s)
	if err != nil {
		return "", nil
	}
	parsed.User = url.UserPassword("", token)
	return parsed.String(), nil
}

func makeInsecureClient(token string) *http.Client {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	return &http.Client{
		Transport: &oauth2.Transport{
			Source: ts,
			Base: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			},
		},
	}
}

func isGitRepoConfiguredViaAnnotation(pr *pipelinesv1alpha1.PipelineRun) bool {
	_, isMapContainsKey := pr.Annotations[gitRepoToReportTo]
	return isMapContainsKey
}

func isGitRevisionConfiguredViaAnnotation(pr *pipelinesv1alpha1.PipelineRun) bool {
	_, isMapContainsKey := pr.Annotations[gitRevision]
	return isMapContainsKey
}
