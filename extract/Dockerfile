FROM registry.access.redhat.com/ubi8/ubi

RUN yum update -y && \
    yum install -y \
      git \
      gcc \
      openssh-clients && \ 
    yum clean all && \
    rm -rf /var/cache/yum

ADD committer.sh /

ADD transform-plugins/ opt 

ADD crane /usr/local/bin

RUN mkdir -p /usr/local/app-root/src && useradd -u 1001 -r -g 0 -d /usr/local/app-root -s /sbin/nologin -c "Default Application User" default && chmod +x /usr/local/bin/crane && chmod +x /opt/*

USER 1001

ENV home /usr/local/app-root

ENTRYPOINT [ "/bin/bash" ]
