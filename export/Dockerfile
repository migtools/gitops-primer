FROM registry.access.redhat.com/ubi8/go-toolset:1.15.14-10 AS crane-builder
RUN git clone https://github.com/konveyor/crane $APP_ROOT/src/github.com/konveyor/crane
WORKDIR $APP_ROOT/src/github.com/konveyor/crane
## Workaround go 1.15 builder. Remove this after go-toolset 1.16 is available
RUN git checkout tags/v0.0.1
RUN sed -i 's/\"os\"/"os"\n\t"io\/ioutil"/g' cmd/apply/apply.go
RUN sed -i 's/os\.ReadFile/ioutil.ReadFile/g' cmd/apply/apply.go
RUN sed -i 's/go 1.16/go 1.15/g' go.mod
## End Workaround go 1.15 builder.
RUN go build -o crane main.go

FROM registry.access.redhat.com/ubi8/go-toolset:1.15.14 AS plugin-builder
RUN mkdir -p $APP_ROOT/src/github.com/konveyor
ADD plugins $APP_ROOT/src/github.com/konveyor/gitops-primer/export/plugins
WORKDIR $APP_ROOT/src/github.com/konveyor/gitops-primer/export/plugins
ENV GOPATH=$APP_ROOT
RUN go get -d ./...
RUN go install ./...

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

COPY --from=crane-builder /opt/app-root/src/github.com/konveyor/crane/crane /usr/local/bin

RUN mkdir -p /usr/local/app-root/src && useradd -u 1001 -r -g 0 -d /usr/local/app-root/src -s /sbin/nologin -c "Default Application User" default && chmod g+rw /usr/local/app-root/src && chmod +x /opt/*

USER 1001

ENV HOME /usr/local/app-root/src

ENTRYPOINT [ "/bin/bash" ]
