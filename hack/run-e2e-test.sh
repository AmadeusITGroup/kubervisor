#!/bin/bash


GIT_ROOT=$(git rev-parse --show-toplevel)
export GOPATH=$GIT_ROOT/../../../../

echo "Start minikube with RBAC option"
minikube start --extra-config=apiserver.Authorization.Mode=RBAC

printf "Waiting for tiller deployment to complete."
until [ $(kubectl get nodes -ojsonpath="{.items[*].metadata.name}") == "minikube" ] > /dev/null 2>&1; do sleep 1; printf "."; done
echo

echo "Create the missing rolebinding for k8s dashboard"
kubectl create clusterrolebinding add-on-cluster-admin --clusterrole=cluster-admin --serviceaccount=kube-system:default

echo "Create the cluster role binding for the helm tiller"
kubectl create clusterrolebinding tiller-cluster-admin  --clusterrole=cluster-admin --serviceaccount=kube-system:default

echo "Init the helm tiller"
helm init

printf "Waiting for tiller deployment to complete."
until [ $(kubectl get deployment -n kube-system tiller-deploy -ojsonpath="{.status.conditions[?(@.type=='Available')].status}") == "True" ] > /dev/null 2>&1; do sleep 1; printf "."; done
echo

eval $(minikube docker-env)
echo "Install the kubervisor operator"

echo "First build the container"
make TAG=latest container

printf  "create and install the kubervisor"
until helm install -n kubervisor charts/kubervisor --wait; do sleep 1; printf "."; done
echo

echo "[[[ Run End2end test ]]] "
cd ./test/e2e && go test -c && ./e2e.test --kubeconfig=$HOME/.kube/config --ginkgo.slowSpecThreshold 260

echo "[[[ Cleaning ]]]"

echo "Remove kubervisor helm chart"
helm del --purge kubervisor

echo "Remove CRD kubervisorservice"
kubectl delete crd kubervisorservices.breaker.kubervisor.io
