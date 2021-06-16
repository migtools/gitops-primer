#! /bin/bash

set -e -o pipefail

kubectl delete deployment test -n test
kubectl delete svc clusterip colors -n test
kubectl delete sa test -n test
