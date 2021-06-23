FROM registry.access.redhat.com/ubi8/ubi

RUN yum update -y && \
    yum install -y \
      git \
      openssh-clients && \ 
    yum clean all && \
    rm -rf /var/cache/yum && \
    curl -sSL "https://golang.org/dl/go1.16.5.linux-amd64.tar.gz" | tar -xz -C /usr/local/bin/

ADD transform-plugins/ /opt/

ADD committer.sh /

RUN git clone https://github.com/konveyor/crane.git && cd crane && /usr/local/bin/go/bin/go build . && mv crane /usr/local/bin/crane && rm -rf ../crane

RUN chmod +x /usr/local/bin/crane && chmod +x /opt/*

ENTRYPOINT [ "/bin/bash" ]
