#!/bin/bash

set -o errexit
set -o nounset
set -o pipefail

./vendor/k8s.io/code-generator/generate-groups.sh all github.com/amadeusitgroup/kubervisor/pkg/client github.com/amadeusitgroup/kubervisor/pkg/api kubervisor:v1alpha1 --go-header-file ./hack/custom-boilerplate.go.txt

