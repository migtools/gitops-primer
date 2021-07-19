#! /bin/bash

set -e -o pipefail

kubectl delete deployment test -n test
kubectl delete svc colors -n test
kubectl delete sa test -n test
kubectl delete extract ci -n test
