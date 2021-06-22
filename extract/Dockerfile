FROM centos:stream8 

RUN yum install -y \
      golang \
      git \
      gcc \
      openssh-clients && \ 
    yum clean all && \
    rm -rf /var/cache/yum

#RUN git clone https://github.com/konveyor/crane.git && cd crane && /usr/local/bin/go/bin/go build . && mv crane /usr/local/bin/crane && rm -rf ../crane
RUN git clone https://github.com/sseago/crane.git && cd crane && git checkout transform-cmd && curl -o cmd/apply/apply.go https://raw.githubusercontent.com/konveyor/crane/e3f60737303385de1235cba489f4f248fe3aa5a2/cmd/apply/apply.go && go build . && mv crane /usr/local/bin/crane && rm -rf ../crane

RUN git clone https://github.com/cooktheryan/crane-lib.git && cd crane-lib/transform/binary-plugin/examples && GOPATH=/opt go install ./... && rm -rf /crane-lib

ADD committer.sh /

ENTRYPOINT [ "/bin/bash" ]
