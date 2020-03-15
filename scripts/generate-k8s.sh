#!/bin/bash

dir=$(pwd)/$(dirname "$0")
GOPATH=$(go env GOPATH)

export GO111MODULE=off
go get k8s.io/code-generator/cmd/client-gen
go get k8s.io/code-generator/cmd/conversion-gen
go get k8s.io/code-generator/cmd/deepcopy-gen
go get k8s.io/code-generator/cmd/defaulter-gen
go get k8s.io/code-generator/cmd/go-to-protobuf
go get k8s.io/code-generator/cmd/import-boss
go get k8s.io/code-generator/cmd/informer-gen
go get k8s.io/code-generator/cmd/lister-gen
go get k8s.io/code-generator/cmd/openapi-gen
go get k8s.io/code-generator/cmd/register-gen
go get k8s.io/code-generator/cmd/set-gen

mkdir -p ${GOPATH}/src/github.com/kuberik
rm -rf ${GOPATH}/src/github.com/kuberik/kuberik
ln -s ${dir}/.. ${GOPATH}/src/github.com/kuberik/kuberik

(
    cd ${GOPATH}/src/k8s.io/code-generator/;
    git reset --hard;
    git checkout v0.17.2;
    git apply ${dir}/code-generator.patch;
    > hack/boilerplate.go.txt;
)

set -e

${GOPATH}/src/k8s.io/code-generator/generate-internal-groups.sh client,defaulter github.com/kuberik/kuberik/pkg/generated "" github.com/kuberik/kuberik/pkg/apis "core:v1alpha1"

mkdir -p pkg/generated
cp -r github.com/kuberik/kuberik/pkg/* pkg/
rm -rf github.com
