FROM registry.access.redhat.com/ubi8/ubi

RUN yum update -y && \
    yum install -y \
      git \
      gcc \
      openssh-clients \
      python3 \
      jq \
    && yum clean all && \
    rm -rf /var/cache/yum && \
    curl -sSL "https://golang.org/dl/go1.16.5.linux-amd64.tar.gz" | tar -xz -C /usr/local/bin && \
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl && \
    rm -rf /kubectl && \
    pip3 install pyyaml

RUN git clone https://github.com/konveyor/crane.git && cd crane && /usr/local/bin/go/bin/go build . && mv crane /usr/local/bin/crane && rm -rf ../crane

ADD committer.sh /

ENTRYPOINT [ "/bin/bash" ]
