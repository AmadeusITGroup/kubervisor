#!/bin/bash
set -o errexit
set -o nounset
set -o pipefail

curl -Lo kubectl https://storage.googleapis.com/kubernetes-release/release/v1.12.0/bin/linux/amd64/kubectl && chmod +x kubectl && sudo mv kubectl /usr/local/bin/
curl -Lo kind https://github.com/kubernetes-sigs/kind/releases/download/0.1.0/kind-linux-amd64 && chmod +x kind && sudo mv kind /usr/local/bin/

GIT_ROOT=$(git rev-parse --show-toplevel)
CREATE_INSECURE_REGISTRY=y CONFIGURE_INSECURE_REGISTRY=y KIND_CONFIG=$GIT_ROOT/hack/kind_config.yaml $GIT_ROOT/hack/create_k8s_cluster.sh