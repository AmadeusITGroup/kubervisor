#!/bin/zsh

GIT_ROOT=$(git rev-parse --show-toplevel)
DEMO_ROOT=$GIT_ROOT/examples/demo
PWD=$(pwd)
KUBERVISOR_PATH=$1
cd $DEMO_ROOT
export GOPATH=$GIT_ROOT/../../../..

export DEMO_NS=demo
minikube start --extra-config=apiserver.Authorization.Mode=RBAC --memory=4096 --kubernetes-version=v1.9.4
minikube update-context
JSONPATH='{range .items[*]}{@.metadata.name}:{range @.status.conditions[*]}{@.type}={@.status};{end}{end}'; until kubectl get nodes -o jsonpath="$JSONPATH" 2>&1 | grep -q "Ready=True"; do sleep 1; done
echo "[ minikube status ]"
minikube status
# add missing role for the dashboard
kubectl create clusterrolebinding add-on-cluster-admin --clusterrole=cluster-admin --serviceaccount=kube-system:default
minikube addons enable ingress

kubectl create -f scripts/tiller-rbac.yaml
helm init --wait --service-account tiller
sleep 60

# install prometheus-operator
helm repo add coreos https://s3-eu-west-1.amazonaws.com/coreos-charts/stable/
helm install --wait coreos/prometheus-operator --name prometheus-operator --version v0.0.19 --namespace $DEMO_NS
helm install --wait coreos/kube-prometheus --name kube-prometheus --version v0.0.53 --namespace $DEMO_NS

# create ingress access
# set *.demo.mk to target your kubernetes ingress endpoint
kubectl apply -f scripts/ingress.yaml -n $DEMO_NS

# Add demo dashboard
kubectl replace -f scripts/kube-grafana-configmap.yaml -n $DEMO_NS

# switch to demo namespaces
kubens $DEMO_NS

eval $(minikube docker-env)
make TAG=latest container

# deploy the Pricer application in version v0.1.0
helm install --wait -n prod-pricer-1a --set deployAppService=true --set config.provider=1a charts/pricer --namespace=$DEMO_NS
# create servicemonitor for the Pricer app
kubectl apply -f $GIT_ROOT/examples/demo/scripts/servicemonitor.yaml -n $DEMO_NS
kubectl apply -f $GIT_ROOT/examples/demo/scripts/servicemonitor.kubervisor.yaml -n $DEMO_NS


# install kubervisor
cd $GIT_ROOT && make TAG=latest container
cd $DEMO_ROOT
helm install --wait -n kubervisor $GIT_ROOT/charts/kubervisor --namespace=$DEMO_NS

