FROM registry.access.redhat.com/ubi8/ubi

RUN yum update -y && \
    yum install -y \
      git \
      gcc \
      openssh-clients && \ 
    yum clean all && \
    rm -rf /var/cache/yum && \
    curl -sSL "https://golang.org/dl/go1.16.5.linux-amd64.tar.gz" | tar -xz -C /usr/local/bin/

ADD transform-plugins/ /opt/

ADD committer.sh /

RUN git clone https://github.com/sseago/crane.git && cd crane && git checkout transform-cmd && curl -o cmd/apply/apply.go https://raw.githubusercontent.com/konveyor/crane/e3f60737303385de1235cba489f4f248fe3aa5a2/cmd/apply/apply.go && /usr/local/bin/go/bin/go build . && mv crane /usr/local/bin/crane && rm -rf ../crane

RUN chmod +x /opt/*

ENTRYPOINT [ "/bin/bash" ]
