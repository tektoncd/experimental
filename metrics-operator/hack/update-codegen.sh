set -o errexit
set -o nounset
set -o pipefail

source $(git rev-parse --show-toplevel)/metrics-operator/hack/setup-temporary-gopath.sh
shim_gopath
trap shim_gopath_clean EXIT


source $(git rev-parse --show-toplevel)/vendor/github.com/tektoncd/plumbing/scripts/library.sh

PREFIX=${GOBIN:-${GOPATH}/bin}

OLDGOFLAGS="${GOFLAGS:-}"
GOFLAGS="-mod=vendor"
# This generates deepcopy,client,informer and lister 
bash ${REPO_ROOT_DIR}/metrics-operator/hack/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/tektoncd/experimental/metrics-operator/pkg/client github.com/tektoncd/experimental/metrics-operator/pkg/apis \
  "monitoring:v1alpha1" \
  --go-header-file ${REPO_ROOT_DIR}/metrics-operator/hack/boilerplate/boilerplate.go.txt

# Knative Injection
# This generates the knative injection packages for the resource package (v1alpha1).
bash ${REPO_ROOT_DIR}/metrics-operator/hack/generate-knative.sh "injection" \
  github.com/tektoncd/experimental/metrics-operator/pkg/client github.com/tektoncd/experimental/metrics-operator/pkg/apis \
  "monitoring:v1alpha1" \
  --go-header-file ${REPO_ROOT_DIR}/metrics-operator/hack/boilerplate/boilerplate.go.txt
GOFLAGS="${OLDGOFLAGS}"

# Make sure our dependencies are up-to-date
${REPO_ROOT_DIR}/metrics-operator/hack/update-deps.sh
