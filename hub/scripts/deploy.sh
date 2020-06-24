#!/usr/bin/env bash
set -e -u -o pipefail

declare -r SCRIPT_DIR=$(cd -P $(dirname "$0") && pwd)


usage() {
  # exit with one if no exit value is provided
  print_usage
  exit ${1:-1}
}

print_usage() {

  read -r -d '' help <<-EOF_HELP || true
Bumps up the image versions of the Tekton Hub services
Usage:
  $( basename $0) VERSION

EOF_HELP

  echo -e "$help"
  return 0
}

patch_image(){
  local deployment=$1; shift
  echo "Patching deployment: $deployment"
  local image=$(oc get deployments/$deployment \
   -o=jsonpath='{ .spec.template.spec.containers[].image }' | cut -f1 -d:)

  oc  --record deployment.apps/$deployment \
    set image deployment/"$deployment" $deployment="$image:$version"
}


main() {
  local version=${1:-''}
  [[ -z "$version" ]] && usage

  local -a deployments=(api validation ui)
  for deployment in ${deployments[@]}; do
    patch_image "$deployment"
  done
  return $?
}

main "$@"
