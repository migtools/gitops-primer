#!/bin/bash
set -e

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
git clone ${REPO} /repo -q
cd /repo
git fetch -q 
existed_in_remote=$(git ls-remote --heads origin ${BRANCH})
if [[ -z ${existed_in_remote} ]]; then
   git checkout -b ${BRANCH}
else  
   git checkout ${BRANCH}
fi
git config --global user.email "${EMAIL}"

# Identify all objects
EXCLUSIONS="pipelinerun|taskrun|images|image.openshift.io|events|machineautoscalers.autoscaling.openshift.io|credentialsrequests.cloudcredential.openshift.io|podnetworkconnectivitychecks.controlplane.operator.openshift.io|leases.coordination.k8s.io|machinehealthchecks.machine.openshift.io|machines.machine.openshift.io|machinesets.machine.openshift.io|baremetalhosts.metal3.io|pods.metrics.k8s.io|alertmanagerconfigs.monitoring.coreos.com|alertmanagers.monitoring.coreos.com|podmonitors.monitoring.coreos.com|volumesnapshots.snapshot.storage.k8s.io|profiles.tuned.openshift.io|tuneds.tuned.openshift.io|endpointslice.discovery.k8s.io|ippools.whereabouts.cni.cncf.io|overlappingrangeipreservations.whereabouts.cni.cncf.io|packagemanifests.packages.operators.coreos.com|endpointslice.discovery.k8s.io|endpoints|pods"

IGNORES="argocd|primer|secret-key|kube-root-ca.crt|image-puller|kubernetes.io/service-account-token|builder|default|default-token|default-dockercfg|deployer|openshift-gitops-operator|redhat-openshift-pipelines-operator|edit|admin|pipeline-dockercfg"


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

export KUBECONFIG=/tmp/kubeconfig
crane export --export-dir /tmp/export
crane transform --export-dir /tmp/export --plugin-dir /opt --transform-dir /tmp/transform
crane apply --export-dir /tmp/export --transform-dir /tmp/transform --output-dir /tmp/outputs 
rm -rf /repo/${NAMESPACE}
cp -rp /tmp/outputs/resources/${NAMESPACE} /repo

git add *
git commit -am 'bot commit'
git push origin ${BRANCH} -q
echo "Merge to ${BRANCH} completed successfully"
