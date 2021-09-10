# Builder Image
FROM golang:1.16.2 as builder

# create and set working directory
RUN mkdir -p /app
WORKDIR /app

# install dependencies
ADD go.mod go.mod
ADD go.sum go.sum
RUN go mod download
ADD . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main main.go

FROM registry.access.redhat.com/ubi8/ubi 
WORKDIR /app
COPY --from=builder /app/main ./main
CMD ["./main"]
