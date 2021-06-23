FROM registry.access.redhat.com/ubi8/ubi

RUN yum update -y && \
    yum install -y \
      git \
      openssh-clients && \ 
    yum clean all && \
    rm -rf /var/cache/yum

ADD transform-plugins/ /opt/

ADD committer.sh /

ADD crane /usr/local/bin

RUN chmod +x /usr/local/bin/crane && chmod +x /opt/*

ENTRYPOINT [ "/bin/bash" ]
