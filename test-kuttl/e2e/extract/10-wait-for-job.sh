#! /bin/bash

set -e -o pipefail

while [[ $(kubectl get po -n test `kubectl get po -n test | grep ci | awk '{print $1}'` -o jsonpath='{.status.containerStatuses[0].state.terminated.reason}') != "Completed" ]]; do
    sleep 1
done
