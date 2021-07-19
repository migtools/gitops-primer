#!/bin/bash
set -e

if [ ${METHOD} == "git" ]; then 
  # Setup SSH
  mkdir -p ~/.ssh/controlmasters
  chmod 711 ~/.ssh
  cp /keys/id_rsa ~/.ssh/id_rsa
  chmod 0600 ~/.ssh/id_rsa
  ssh-keyscan -t rsa github.com >> ~/.ssh/known_hosts

cat - <<SSHCONFIG > ~/.ssh/config
Host *
  # Wait max 30s to establish connection
  ConnectTimeout 30
  # Control persist to speed 2nd ssh connection
  ControlMaster auto
  ControlPath ~/.ssh/controlmasters/%C
  ControlPersist 5
  # Disables warning when IP is added to known_hosts
  CheckHostIP no
  # Use the identity provided via attached Secret
  IdentityFile /keys/id_rsa
  # Enable protocol-level keepalive to detect connection failure
  ServerAliveCountMax 4
  ServerAliveInterval 30
  # Using protocol-level, so we don't need TCP-level
  TCPKeepAlive no
SSHCONFIG

  # Setup the repository
  rm -rf /output/*
  git clone ${REPO} /output/repo -q
  cd /output/repo
  git fetch -q 
  existed_in_remote=$(git ls-remote --heads origin ${BRANCH})
  if [[ -z ${existed_in_remote} ]]; then
     git checkout -b ${BRANCH}
  else  
     git checkout ${BRANCH}
  fi
  git config --global user.email "${EMAIL}"
fi

TOKEN=`cat /var/run/secrets/kubernetes.io/serviceaccount/token | base64 -w0`
CA=`cat /var/run/secrets/kubernetes.io/serviceaccount/ca.crt |base64 -w0`

# Generate KUBECONFIG
echo "
apiVersion: v1
kind: Config
clusters:
- name: default-cluster
  cluster:
    certificate-authority-data: ${CA}
contexts:
- name: default-context
  context:
    cluster: default-cluster
    namespace: ${NAMESPACE}
    user: default-user
current-context: default-context
users:
- name: default-user
  user:
    token: ${TOKEN}
" > /tmp/kubeconfig

if [ ${METHOD} == "download" ]; then
  mkdir /output/repo
  cd /output/repo
fi

export KUBECONFIG=/tmp/kubeconfig
crane export -e /tmp/export
crane transform -e /tmp/export/resources -p /opt -t /tmp/transform
crane apply -e /tmp/export/resources -t /tmp/transform --output-dir /output/repo


if [ ${METHOD} == "git" ]; then 
  git add *
  git commit -am 'bot commit'
  git push origin ${BRANCH} -q
  echo "Merge to ${BRANCH} completed successfully"
else
  zip /output/export.zip ${NAMESPACE}
fi
