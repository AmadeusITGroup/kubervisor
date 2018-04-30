#!/bin/bash

DEMO_ROOT=$(git rev-parse --show-toplevel)/examples/demo
. ${DEMO_ROOT}/scripts/demo-utils.sh

PWD=$(pwd)
cd $DEMO_ROOT

desc "Let see what is running"
run "kubectl get pods"

desc "run ./scripts/readprice.sh in another terminal"
desc "run 'watch -n1 kubectl get pods,svc -l app=pricer --show-labels'in another terminal"

desc "now we create the KubervisorService for the app"
run "cat scripts/pricer-kubervisorservice_1.yaml"
run "kubectl apply -f scripts/pricer-kubervisorservice_1.yaml"

desc "add kubervisor label on the service"
desc 'kubectl patch svc pricer-1a --type=json -p=[{"op": "add", "path": "/spec/selector", "value": {"app":"pricer","kubervisor/traffic":"yes" } }]'
kubectl patch svc pricer-1a --type=json -p='[{"op": "add", "path": "/spec/selector", "value": {"app":"pricer","kubervisor/traffic":"yes" } }]'
run "kubectl get svc pricer-1a -o yaml"

desc "update the configuration of one pod in order to generates bad responses"
run "kubectl exec -t $(kubectl get pod -l app=pricer --output=jsonpath={.items[0].metadata.name}) -- curl -X POST http://localhost:8080/setconfig?kmprice=0"

desc "lets see the KubervisorService status"
run "watch kubectl plugin kubervisor"

desc "kill the pod with anomaly"
run "kubectl delete pod $(kubectl get pod -l app=pricer --output=jsonpath={.items[0].metadata.name})"

desc "change the KubervisorService configuration"
run "cat scripts/pricer-kubervisorservice_2.yaml"
run "kubectl apply -f scripts/pricer-kubervisorservice_2.yaml"

desc "update the configuration of one pod in order to generates bad responses"
run "kubectl exec -t $(kubectl get pod -l app=pricer --output=jsonpath={.items[0].metadata.name}) -- curl -X POST http://localhost:8080/setconfig?kmprice=0"

desc "lets see the KubervisorService status"
run "watch kubectl plugin kubervisor"

desc "kill the pod with anomaly"
run "kubectl delete pod $(kubectl get pod -l app=pricer --output=jsonpath={.items[0].metadata.name})"

desc "change again the KubervisorService configuration"
run "cat scripts/pricer-kubervisorservice_3.yaml"
run "kubectl apply -f scripts/pricer-kubervisorservice_3.yaml"

desc "update the configuration of one pod in order to generates bad responses"
run "kubectl exec -t $(kubectl get pod -l app=pricer --output=jsonpath={.items[0].metadata.name}) -- curl -X POST http://localhost:8080/setconfig?kmprice=0"

desc "lets see the KubervisorService status"
run "watch kubectl plugin kubervisor"