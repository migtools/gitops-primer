# Builder Image
FROM registry.access.redhat.com/ubi8/go-toolset:1.17 as builder

# create and set working directory
RUN mkdir -p $APP_ROOT/src/github.com/migtools/gitops-primer/webhook
WORKDIR $APP_ROOT/src/github.com/migtools/gitops-primer/webhook

# install dependencies
ADD go.mod go.mod
ADD go.sum go.sum
RUN go mod download
ADD . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main main.go

FROM registry.access.redhat.com/ubi8/ubi
WORKDIR /app
COPY --from=builder /opt/app-root/src/github.com/migtools/gitops-primer/webhook/main ./main
CMD ["./main"]
