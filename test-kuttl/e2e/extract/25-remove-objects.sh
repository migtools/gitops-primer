#! /bin/bash

set -e -o pipefail

kubectl get extract -n test -o yaml
kubectl get secrets -n test
kubectl get po -n test
kubectl get po -n gitops-primer-system
sleep 2m
kubectl get po -n gitops-primer-system
kubectl logs -n gitops-primer-system `kubectl get po -n gitops-primer-system | grep primer | awk '{print $1}' | head -n1` --all-containers
sleep 3m
kubectl logs -n test `kubectl get po -n test | grep primer | awk '{print $1}' | head -n1`
