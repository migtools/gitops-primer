#! /bin/bash
set -e -o pipefail

kubectl apply -f - <<EOF
---
apiVersion: primer.gitops.io/v1alpha1
kind: Extract
metadata:
  name: ci
spec:
  repo: git@github.com:cooktheryan/primer-poc.git
  branch: ci
  action: merge
  secret: secret-key
EOF
