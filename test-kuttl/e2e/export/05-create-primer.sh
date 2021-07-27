#! /bin/bash
set -e -o pipefail

kubectl apply -n test -f - <<EOF
---
apiVersion: primer.gitops.io/v1alpha1
kind: Export
metadata:
  name: ci
spec:
  repo: git@github.com:cooktheryan/primer-poc.git
  branch: ci
  email: nobody@everybody.com
  secret: secret-key
  method: git
EOF
