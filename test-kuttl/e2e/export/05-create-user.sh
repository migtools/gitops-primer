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

kubectl apply -n test -f - <<EOF
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: primer
rules:
- apiGroups:
  - primer.gitops.io
  resources:
  - exports
  verbs:
  - create
  - delete
  - get
  - list
EOF

kubectl apply -n test -f - <<EOF
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: primer-binding 
subjects:
- kind: User 
  name: bob
  apiGroup: rbac.authorization.k8s.io
roleRef:
  kind: ClusterRole
  name: primer
  apiGroup: rbac.authorization.k8s.io
EOF
