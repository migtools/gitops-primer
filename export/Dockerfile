FROM registry.access.redhat.com/ubi8/go-toolset:1.17.10-4 AS crane-builder
RUN git clone https://github.com/migtools/crane $APP_ROOT/src/github.com/migtools/crane
WORKDIR $APP_ROOT/src/github.com/migtools/crane
RUN git checkout tags/v0.0.5
RUN go build -o crane main.go

FROM registry.access.redhat.com/ubi8/go-toolset:1.17.10-4 AS plugin-builder
RUN mkdir -p $APP_ROOT/src/github.com/migtools
RUN curl -L -o go.mod https://raw.githubusercontent.com/migtools/gitops-primer/main/go.mod
ADD plugins $APP_ROOT/src/github.com/migtools/gitops-primer/export/plugins
WORKDIR $APP_ROOT/src/github.com/migtools/gitops-primer/export/plugins
ENV GOPATH=$APP_ROOT
RUN go get -d ./...
RUN go install ./...

FROM registry.access.redhat.com/ubi8/ubi-minimal

RUN microdnf update -y && \
    microdnf install -y \
      git \
      gcc \
      zip \
      openssh-clients && \
    microdnf clean all && \
    rm -rf /var/cache/yum

ADD committer.sh /

COPY --from=plugin-builder /opt/app-root/bin /opt/transform-plugins

COPY --from=crane-builder /opt/app-root/src/github.com/migtools/crane/crane /usr/local/bin

RUN mkdir -p /usr/local/app-root/src && useradd -u 1001 -r -g 0 -d /usr/local/app-root/src -s /sbin/nologin -c "Default Application User" default && chmod g+rw /usr/local/app-root/src && chmod +x /opt/*

USER 1001

ENV HOME /usr/local/app-root/src

ENTRYPOINT [ "/bin/bash" ]
