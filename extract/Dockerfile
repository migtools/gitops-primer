FROM registry.access.redhat.com/ubi8/ubi

RUN yum update -y && \
    yum install -y \
      git \
      gcc \
      openssh-clients && \ 
    yum clean all && \
    rm -rf /var/cache/yum && \
    curl -sSL "https://golang.org/dl/go1.16.5.linux-amd64.tar.gz" | tar -xz -C /usr/local/bin/ && \
    curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl" && \
    install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl && \
    rm -rf /kubectl

#RUN git clone https://github.com/konveyor/crane.git && cd crane && /usr/local/bin/go/bin/go build . && mv crane /usr/local/bin/crane && rm -rf ../crane
RUN git clone https://github.com/sseago/crane.git && cd crane && curl -o cmd/apply/apply.go https://raw.githubusercontent.com/konveyor/crane/e3f60737303385de1235cba489f4f248fe3aa5a2/cmd/apply/apply.go && git checkout transform-cmd && /usr/local/bin/go/bin/go build . && mv crane /usr/local/bin/crane && rm -rf ../crane

RUN git clone https://github.com/cooktheryan/crane-lib.git && cd crane-lib/transform/binary-plugin/examples && GOPATH=/opt /usr/local/bin/go/bin/go install ./... && rm -rf ../../../crane-lib

ADD committer.sh /

ENTRYPOINT [ "/bin/bash" ]
