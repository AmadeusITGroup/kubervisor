#!/bin/zsh

export DEMO_NS=demo
kubectl -n $DEMO_NS delete kubervisorservice pricer-1a
kubectl -n $DEMO_NS patch svc pricer-1a --type='json' -p='[{"op": "add", "path": "/spec/selector", "value": {"app":"pricer"} }]'
kubectl -n $DEMO_NS delete pod -l app=pricer
