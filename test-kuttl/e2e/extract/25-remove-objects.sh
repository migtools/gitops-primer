#! /bin/bash

set -e -o pipefail

kubectl delete deployment test -n test
kubectl delete svc colors -n test
kubectl delete sa test -n test
kubectl get extract -n test -o yaml
kubectl get po -n test
sleep 3m
kubectl logs -n test `kubectl get po -n test | grep primer | awk '{print $1}' | head -n1`
