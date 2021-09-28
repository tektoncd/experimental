package resolution

import (
	"errors"
	"fmt"

	"github.com/tektoncd/pipeline/pkg/apis/pipeline/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	GitRemoteRepoAnnotationKey = "remote-resolution-git-pipeline-ref-repo"
	GitRemoteSHAAnnotationKey  = "remote-resolution-git-pipeline-ref-sha"
	GitRemotePathAnnotationKey = "remote-resolution-git-pipeline-ref-path"
)

var (
	MissingGitAnnotations = errors.New("no git reference in annotations")
	InvalidGitAnnotations = errors.New("invalid git reference in annotations")
)

type MissingGitAnnotation struct {
	missingAnnotation string
}

func (e *MissingGitAnnotation) Error() string {
	return fmt.Sprintf("missing annotation %q", e.missingAnnotation)
}

func parseGitAnnotations(annotations map[string]string) (GitResolutionRequest, error) {
	req := GitResolutionRequest{}

	if annotations != nil {
		if val, ok := annotations[GitRemoteRepoAnnotationKey]; ok {
			req.Repo = val
		} else {
			return req, &MissingGitAnnotation{missingAnnotation: GitRemoteRepoAnnotationKey}
		}

		if val, ok := annotations[GitRemoteSHAAnnotationKey]; ok {
			req.SHA = val
		} else {
			return req, &MissingGitAnnotation{missingAnnotation: GitRemoteSHAAnnotationKey}
		}

		if val, ok := annotations[GitRemotePathAnnotationKey]; ok {
			req.Path = val
		} else {
			return req, &MissingGitAnnotation{missingAnnotation: GitRemotePathAnnotationKey}
		}

		if err := req.Validate(); err != nil {
			return req, err
		}

		return req, nil
	}
	return req, &MissingGitAnnotation{missingAnnotation: "all"}
}

type GitResolutionRequest struct {
	Repo string
	SHA  string
	Path string
}

func (req *GitResolutionRequest) Validate() error {
	// TODO(sbwsg): add validation of repo url and path string.
	// url must be fully formed
	// path must be relative and not include ..
	// Assume SHA can look like anything for now.
	return nil
}

func (req *GitResolutionRequest) Execute() (*metav1.ObjectMeta, *v1beta1.PipelineSpec, error) {
	// `git clone $repo ./tmpdir`
	// `cd ./tmpdir`
	// `cat $path`
	// if tempDir, err := os.MkdirTemp(); err != nil {
	// 	return nil, nil, fmt.Errorf("error creating temp dir for repo: %v", err)
	// } else {
	// 	yamlPath := filepath.Join(tempDir, req.Path)
	// 	cloneCmd := exec.Command("git", "clone", req.Repo, tempDir)
	// 	cloneCmd.CombinedOutput()
	// 	b, err := os.ReadFile(yamlPath)
	// }
	return nil, nil, nil
}
