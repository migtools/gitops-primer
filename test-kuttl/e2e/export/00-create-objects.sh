#! /bin/bash

set -e -o pipefail

kubectl create deployment test --image nginx -n test
kubectl create svc clusterip colors --tcp 8080 -n test
kubectl create sa test -n test
