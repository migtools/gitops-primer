# Build the manager binary
FROM registry.access.redhat.com/ubi8/go-toolset:1.15.14 as builder

RUN mkdir -p $APP_ROOT/src/github.com/migtools/gitops-primer
WORKDIR $APP_ROOT/src/github.com/migtools/gitops-primer
# Copy the Go Modules manifests
COPY go.mod go.mod
COPY go.sum go.sum
# cache deps before building and copying source so that we don't need to re-download as much
# and so that source changes don't invalidate our downloaded layer
RUN go mod download

# Copy the go source
COPY main.go main.go
COPY api/ api/
COPY controllers/ controllers/

# Build
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 GO111MODULE=on go build -a -o manager main.go

# Use distroless as minimal base image to package the manager binary
# Refer to https://github.com/GoogleContainerTools/distroless for more details
FROM registry.access.redhat.com/ubi8/ubi-minimal
WORKDIR /
COPY --from=builder /opt/app-root/src/github.com/migtools/gitops-primer/manager /manager
USER 65532:65532
ENTRYPOINT ["/manager"]
