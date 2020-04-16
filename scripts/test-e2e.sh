#!/bin/bash

pwd

generated_dir=$(mktemp -d)
function cleanup {
    rm -rf "${generated_dir}"
}
trap cleanup EXIT

kustomize build deploy/crds > ${generated_dir}/crds.yaml
kustomize build deploy/operator > ${generated_dir}/operator.yaml

operator-sdk test local ./test/e2e \
    --namespaced-manifest ${generated_dir}/operator.yaml \
    --global-manifest ${generated_dir}/crds.yaml \
    --namespace default \
    --up-local \
