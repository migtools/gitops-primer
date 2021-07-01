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

RUN chmod +x /usr/local/bin/crane && chmod +x /opt/* && chmod g+rw /opt/app-root/src

USER 1001

WORKDIR /opt/app-root/src

ENTRYPOINT [ "/bin/bash" ]
