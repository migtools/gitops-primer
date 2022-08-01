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

TOKEN=`cat /var/run/secrets/kubernetes.io/serviceaccount/token`
CA=`cat /var/run/secrets/kubernetes.io/serviceaccount/ca.crt |base64 -w0`

# Generate KUBECONFIG
echo "
apiVersion: v1
kind: Config
clusters:
  - name: mycluster
    cluster:
      certificate-authority-data: ${CA}
      server: https://${KUBERNETES_SERVICE_HOST}:${KUBERNETES_SERVICE_PORT}
contexts:
  - name: primer-export-primer@mycluster
    context:
      cluster: mycluster
      namespace: ${NAMESPACE} 
      user: primer-export-primer
users:
  - name: primer-export-primer
    user:
      token: ${TOKEN}
current-context: primer-export-primer@mycluster
" > /tmp/kubeconfig

if [ ${METHOD} == "download" ]; then
  mkdir -p /output/repo
  cd /output/repo
fi

export KUBECONFIG=/tmp/kubeconfig
if [ -z "$GROUP" ]; then crane export --export-dir /tmp/export --as ${USER} ; else IFS=';'; read -ra GARR <<< "${GROUP}"; crane export --export-dir /tmp/export --as ${USER} --as-group ${GARR[@]}; fi
crane transform --export-dir /tmp/export/resources --plugin-dir /opt --transform-dir /tmp/transform --skip-plugins KubernetesPlugin
crane apply --export-dir /tmp/export/resources --transform-dir /tmp/transform --output-dir /output/repo


if [ ${METHOD} == "git" ]; then 
  if [[ $(git status -s) ]]; then
     git add *
     git commit -am 'bot commit'
     git push origin ${BRANCH} -q
     echo "Merge to ${BRANCH} completed successfully"
  else
     exit 0
  fi
else
  cd /output/repo
  zip -r /output/${NAMESPACE}-${TIME} ${NAMESPACE}
  rm -rf /output/repo
fi

