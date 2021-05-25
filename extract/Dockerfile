FROM registry.access.redhat.com/ubi8/ubi

RUN yum update -y && \
    yum install -y \
      git \
      openssh-clients \
      python3 \
      jq \
    && yum clean all && \
    rm -rf /var/cache/yum && \
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl && \
    rm -rf /kubectl && \
    pip3 install pyyaml


ADD committer.sh /

USER 1001

ENTRYPOINT [ "/bin/bash" ]
