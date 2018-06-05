#!/usr/bin/env bash

KUBE_CONFIG_PATH=$HOME"/.kube"
if [ -n "$1" ]
then KUBE_CONFIG_PATH=$1
fi

KUBERVISOR_PLUGIN_BIN_NAME="kubectl-plugin"
KUBERVISOR_PLUGIN_PATH=$KUBE_CONFIG_PATH/plugins/kubervisor

mkdir -p $KUBERVISOR_PLUGIN_PATH

GIT_ROOT=$(git rev-parse --show-toplevel)
cp $GIT_ROOT/bin/$KUBERVISOR_PLUGIN_BIN_NAME $KUBERVISOR_PLUGIN_PATH/$KUBERVISOR_PLUGIN_BIN_NAME

cat > $KUBERVISOR_PLUGIN_PATH/plugin.yaml << EOF1
name: "kubervisor"
shortDesc: "kubervisor shows kubervisor custom resources"
longDesc: >
  kubervisor shows kubervisor custom resources
command: ./kubectl-plugin
flags:
- name: "ks"
  shorthand: "k"
  desc: "KuverisorService name"
  defValue: ""
EOF1
