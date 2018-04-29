#!/usr/bin/env bash

KUBE_CONFIG_PATH=$HOME"/.kube"
if [ -n "$1" ]
then KUBE_CONFIG_PATH=$1
fi

REDIS_PLUGIN_BIN_NAME="kubectl-plugin"
REDIS_PLUGIN_PATH=$KUBE_CONFIG_PATH/plugins/kubervisor

mkdir -p $REDIS_PLUGIN_PATH

GIT_ROOT=$(git rev-parse --show-toplevel)
cp $GIT_ROOT/bin/$REDIS_PLUGIN_BIN_NAME $REDIS_PLUGIN_PATH/$REDIS_PLUGIN_BIN_NAME

cat > $REDIS_PLUGIN_PATH/plugin.yaml << EOF1
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
