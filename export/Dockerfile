FROM quay.io/konveyor/builder:latest AS crane-builder
RUN git clone https://github.com/konveyor/crane $APP_ROOT/src/github.com/konveyor/crane
WORKDIR $APP_ROOT/src/github.com/konveyor/crane
RUN git checkout tags/v0.0.1
RUN go build -o crane main.go

FROM registry.access.redhat.com/ubi8/go-toolset:1.15.14 AS plugin-builder
RUN mkdir -p $APP_ROOT/src/github.com/konveyor
ADD plugins $APP_ROOT/src/github.com/konveyor/gitops-primer/export/plugins
WORKDIR $APP_ROOT/src/github.com/konveyor/gitops-primer/export/plugins
ENV GOPATH=$APP_ROOT
RUN go get -d ./...
RUN go install ./...

FROM quay.io/konveyor/builder:latest AS crane-lib-plugin-builder
RUN git clone https://github.com/konveyor/crane-lib $APP_ROOT/src/github.com/konveyor/crane-lib
WORKDIR $APP_ROOT/src/github.com/konveyor/crane-lib
RUN git checkout tags/v0.0.1
ENV GOPATH=$APP_ROOT
RUN go get -d ./...
RUN go install ./transform/binary-plugin/examples/openshift/


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

COPY --from=plugin-builder /opt/app-root/bin /opt/transform-plugins

COPY --from=crane-lib-plugin-builder /opt/app-root/bin /opt/transform-plugins

COPY --from=crane-builder /opt/app-root/src/github.com/konveyor/crane/crane /usr/local/bin

RUN mkdir -p /usr/local/app-root/src && useradd -u 1001 -r -g 0 -d /usr/local/app-root/src -s /sbin/nologin -c "Default Application User" default && chmod g+rw /usr/local/app-root/src && chmod +x /opt/*

USER 1001

ENV HOME /usr/local/app-root/src

ENTRYPOINT [ "/bin/bash" ]
