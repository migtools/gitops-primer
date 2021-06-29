FROM registry.access.redhat.com/ubi8/ubi

RUN yum update -y && \
    yum install -y \
      git \
      gcc \
      openssh-clients && \ 
    yum clean all && \
    rm -rf /var/cache/yum && \
    curl -sSL "https://golang.org/dl/go1.16.5.linux-amd64.tar.gz" | tar -xz -C /usr/local/bin/

ADD committer.sh /

ADD transform-plugins/ opt 

ADD crane /usr/local/bin

RUN chmod +x /usr/local/bin/crane && chmod +x /opt/*

ENTRYPOINT [ "/bin/bash" ]
