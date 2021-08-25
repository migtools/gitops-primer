#! /bin/bash
set -e -o pipefail

openssl req -new -newkey rsa:4096 -nodes -keyout bob-k8s.key -out bob-k8s.csr -subj "/CN=bob/O=devops"
SECRET=`cat bob-k8s.csr | base64 | tr -d '\n'`

kubectl apply -n test -f - <<EOF
---
apiVersion: certificates.k8s.io/v1beta1
kind: CertificateSigningRequest
metadata:
  name: bob-k8s-access
spec:
  groups:
  - system:authenticated
  request: ${SECRET} 
  usages:
  - client auth
EOF

kubectl certificate approve bob-k8s-access

kubectl label ns test user=bob env=sandbox
kubectl create rolebinding bob-admin --namespace=test --clusterrole=admin --user=bob
