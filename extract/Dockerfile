FROM registry.access.redhat.com/ubi8/ubi

RUN yum update -y && \
    yum install -y \
      git \
      gcc \
      openssh-clients && \ 
    yum clean all && \
    rm -rf /var/cache/yum

ADD committer.sh /

ADD transform-plugins opt/transform-plugins

ADD crane /usr/local/bin

RUN mkdir -p /opt/app-root/src && useradd -u 1001 -r -g 0 -d /opt/app-root/src -s /sbin/nologin -c "Default Application User" default && chmod +x /usr/local/bin/crane && chmod +x /opt/* && chmod g+rw /opt/app-root/src

USER 1001

ENV HOME /opt/app-root/src

ENTRYPOINT [ "/bin/bash" ]
