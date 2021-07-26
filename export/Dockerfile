FROM registry.access.redhat.com/ubi8/ubi

RUN yum update -y && \
    yum install -y \
      git \
      gcc \
      zip \
      openssh-clients && \ 
    yum clean all && \
    rm -rf /var/cache/yum

ADD committer.sh /

ADD transform-plugins/ opt 

ADD crane /usr/local/bin

RUN mkdir -p /usr/local/app-root/src && useradd -u 1001 -r -g 0 -d /usr/local/app-root/src -s /sbin/nologin -c "Default Application User" default && chmod g+rw /usr/local/app-root/src && chmod +x /usr/local/bin/crane && chmod +x /opt/*

USER 1001

ENV HOME /usr/local/app-root/src

ENTRYPOINT [ "/bin/bash" ]
