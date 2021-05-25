
#! /bin/bash

set -e -o pipefail

POD=`kubectl get po -n test | grep ci | awk '{print $1}'`
while [[ $(kubectl get po -n test $POD -o jsonpath='{.status.containerStatuses[0].state.terminated.reason}') != "Completed" ]]; do
    sleep 1
done
