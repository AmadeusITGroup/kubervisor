#!/usr/bin/env bash

# Copyright 2018 The Kubernetes Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# This script handles the creation of multiple clusters using kind and the
# ability to create and configure an insecure container registry.

# Mostly copy/paste from https://github.com/kubernetes-sigs/federation-v2/scripts/create-clusters.sh
# - Add support of Kind config file, to allow multi nodes cluster
# - Create only on cluster by default

set -o errexit
set -o nounset
set -o pipefail

source "$(dirname "${BASH_SOURCE}")/util.sh"
CREATE_INSECURE_REGISTRY="${CREATE_INSECURE_REGISTRY:-}"
CONFIGURE_INSECURE_REGISTRY="${CONFIGURE_INSECURE_REGISTRY:-}"
CONTAINER_REGISTRY_HOST="${CONTAINER_REGISTRY_HOST:-172.17.0.1:5000}"
NUM_CLUSTERS="${NUM_CLUSTERS:-1}"
KIND_CONFIG="${KIND_CONFIG:-}"
docker_daemon_config="/etc/docker/daemon.json"
kubeconfig="${HOME}/.kube/config"

function create-insecure-registry() {
  # Run insecure registry as container
  docker run -d -p 5000:5000 --restart=always --name registry registry:2
}

function configure-insecure-registry() {
  local err=
  if sudo test -f "${docker_daemon_config}"; then
    echo <<EOF "Error: ${docker_daemon_config} exists and \
CONFIGURE_INSECURE_REGISTRY=${CONFIGURE_INSECURE_REGISTRY}. This script needs \
to add an 'insecure-registries' entry with host '${CONTAINER_REGISTRY_HOST}' to \
${docker_daemon_config}. Please make the necessary changes or backup and try again."
EOF
    err=true
  elif pgrep -a dockerd | grep -q 'insecure-registry'; then
    echo <<EOF "Error: CONFIGURE_INSECURE_REGISTRY=${CONFIGURE_INSECURE_REGISTRY} \
and about to write ${docker_daemon_config}, but dockerd is already configured with \
an 'insecure-registry' command line option. Please make the necessary changes or disable \
the command line option and try again."
EOF
    err=true
  fi

  if [[ "${err}" ]]; then
    if [[ "${CREATE_INSECURE_REGISTRY}" ]]; then
      docker kill registry &> /dev/null
      docker rm registry &> /dev/null
    fi
    return 1
  fi

  configure-insecure-registry-and-reload "sudo bash -c" $(pgrep dockerd)
}

function configure-insecure-registry-and-reload() {
  local cmd_context="${1}" # context to run command e.g. sudo, docker exec
  local docker_pid="${2}"
  ${cmd_context} "$(insecure-registry-config-cmd)"
  ${cmd_context} "$(reload-docker-daemon-cmd "${docker_pid}")"
}

function insecure-registry-config-cmd() {
  echo "cat <<EOF > ${docker_daemon_config}
{
    \"insecure-registries\": [\"${CONTAINER_REGISTRY_HOST}\"]
}
EOF
"
}

function reload-docker-daemon-cmd() {
  echo "kill -SIGHUP ${1}"
}

function create-clusters() {
  local num_clusters=${1}

  for i in $(seq ${num_clusters}); do
    # kind will create cluster with name: kind-${i}
    kind create cluster --name ${i} --wait 4m ${2:+--config $2}

    # TODO: Configure insecure registry on kind host cluster. Remove once
    # https://github.com/kubernetes-sigs/kind/issues/110 is resolved.
    echo "Configuring insecure container registry on kind host cluster name: ${1}"
    configure-insecure-registry-on-cluster ${i}
  done


}

function fixup-cluster() {
  local i=${1} # cluster num

  local kubeconfig_path="$(kind get kubeconfig-path --name ${i})"
  export KUBECONFIG="${KUBECONFIG:-}:${kubeconfig_path}"

  # TODO(font): Need to set container IP address in order for clusters to reach
  # kube API servers in other clusters until
  # https://github.com/kubernetes-sigs/kind/issues/111 is resolved.
  local container_ip_addr=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' kind-${i}-control-plane)
  sed -i "s/localhost/${container_ip_addr}/" ${kubeconfig_path}

  # TODO(font): Need to rename auth user name to avoid conflicts when using
  # multiple cluster kubeconfigs. Remove once
  # https://github.com/kubernetes-sigs/kind/issues/112 is resolved.
  sed -i "s/kubernetes-admin/kubernetes-kind-${i}-admin/" ${kubeconfig_path}
}

function configure-insecure-registry-on-cluster() {
  for i in $(docker ps --format '{{.Names}}' | grep "kind-${1}"); do
    echo "configure insecure registry on node: ${i}"
    configure-insecure-registry-and-reload "docker exec ${i} bash -c" '$(pgrep dockerd)'
  done
}

if [[ "${CREATE_INSECURE_REGISTRY}" ]]; then
  echo "Creating container registry on host"
  create-insecure-registry
fi

if [[ "${CONFIGURE_INSECURE_REGISTRY}" ]]; then
  echo "Configuring container registry on host"
  configure-insecure-registry
fi

echo "Creating ${NUM_CLUSTERS} clusters"
create-clusters ${NUM_CLUSTERS} ${KIND_CONFIG}

echo "Complete"
