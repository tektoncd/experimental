set -o errexit
set -o nounset
set -o pipefail

source $(git rev-parse --show-toplevel)/concurrency/hack/setup-temporary-gopath.sh
shim_gopath
trap shim_gopath_clean EXIT

source $(git rev-parse --show-toplevel)/concurrency/vendor/github.com/tektoncd/plumbing/scripts/library.sh

PREFIX=${GOBIN:-${GOPATH}/bin}

OLDGOFLAGS="${GOFLAGS:-}"
GOFLAGS="-mod=vendor"
# generate the code with:
# --output-base    because this script should also be able to run inside the vendor dir of
#                  k8s.io/kubernetes. The output-base is needed for the generators to output into the vendor dir
#                  instead of the $GOPATH directly. For normal projects this can be dropped.
# This generates deepcopy,client,informer and lister for the resource package (v1alpha1)
# This is separate from the pipeline package as resource are staying in v1alpha1 and they
# need to be separated (at least in terms of go package) from the pipeline's packages to
# not having dependency cycle.
# This generates deepcopy,client,informer and lister for the pipeline package (v1alpha1 and v1beta1)
bash ${REPO_ROOT_DIR}/concurrency/hack/generate-groups.sh "deepcopy,client,informer,lister" \
  github.com/tektoncd/experimental/concurrency/pkg/client github.com/tektoncd/experimental/concurrency/pkg/apis \
  "concurrency:v1alpha1" \
  --go-header-file ${REPO_ROOT_DIR}/concurrency/hack/boilerplate/boilerplate.go.txt

# Knative Injection
# This generates the knative injection packages for the resource package (v1alpha1).
bash ${REPO_ROOT_DIR}/concurrency/hack/generate-knative.sh "injection" \
  github.com/tektoncd/experimental/concurrency/pkg/client github.com/tektoncd/experimental/concurrency/pkg/apis \
  "concurrency:v1alpha1" \
  --go-header-file ${REPO_ROOT_DIR}/concurrency/hack/boilerplate/boilerplate.go.txt
GOFLAGS="${OLDGOFLAGS}"

# Depends on generate-groups.sh to install bin/deepcopy-gen
# TODO: this doesn't work
${PREFIX}/deepcopy-gen \
  -O zz_generated.deepcopy \
  --go-header-file ${REPO_ROOT_DIR}/concurrency/hack/boilerplate/boilerplate.go.txt \
  -i github.com/tektoncd/experimental/concurrency/pkg/apis/concurrency/v1alpha1

# Make sure our dependencies are up-to-date
${REPO_ROOT_DIR}/concurrency/hack/update-deps.sh