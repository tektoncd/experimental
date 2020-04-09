#!/usr/bin/env sh

# This script is meant to be run inside a container running jnorwood/helm-docs image

apk add -u git

helm-docs

OUTOFDATE=$(git status --porcelain)

if [ ! -z "$OUTOFDATE" ]; then
    echo "helm charts doc is out of date, please run `helm-docs` to generate the new docs before pushing your changes"

    exit 1
fi

exit 0
