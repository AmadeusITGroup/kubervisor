#!/bin/bash


GIT_ROOT=$(git rev-parse --show-toplevel)
export GOPATH=$GIT_ROOT/../../../../

if [[ $SKIPINIT != 1 ]]; then

    echo "Start minikube with RBAC option"
    kind create cluster --config $GIT_ROOT/hack/kind_config.yaml --wait 3m
    export KUBECONFIG="$(kind get kubeconfig-path)"

    echo "Create the missing rolebinding for k8s dashboard"
    kubectl create clusterrolebinding add-on-cluster-admin --clusterrole=cluster-admin --serviceaccount=kube-system:default

    echo "Create the cluster role binding for the helm tiller"
    kubectl create clusterrolebinding tiller-cluster-admin  --clusterrole=cluster-admin --serviceaccount=kube-system:default

    echo "Init the helm tiller"
    helm init

    printf "Waiting for tiller deployment to complete."
    until [ $(kubectl get deployment -n kube-system tiller-deploy -ojsonpath="{.status.conditions[?(@.type=='Available')].status}") == "True" ] > /dev/null 2>&1; do sleep 1; printf "."; done
    echo
fi

if [[ $SKIPMOCK != 1 ]]; then
    cd $GIT_ROOT/test/e2e
    make TAG=latest container
    cd -
fi

echo "Install the kubervisor operator"

echo "First build the container"
make TAG=latest container

printf  "create and install the kubervisor"
until helm install -n kubervisor charts/kubervisor --wait; do sleep 1; printf "."; done
echo

echo "[[[ Run End2end test ]]] "
cd ./test/e2e && go test -c && ./e2e.test --kubeconfig=$KUBECONFIG --ginkgo.slowSpecThreshold 260

echo "[[[ Cleaning ]]]"

echo "Remove kubervisor helm chart"
helm del --purge kubervisor

echo "Remove CRD kubervisorservice"
kubectl delete crd kubervisorservices.kubervisor.k8s.io
