FROM golang:latest AS builder

# Install protoc
RUN apt-get update && apt-get install -y unzip
RUN curl -L -o protoc.zip https://github.com/protocolbuffers/protobuf/releases/download/v27.1/protoc-27.1-linux-x86_64.zip \
    && unzip protoc.zip -d /usr/local/ && rm protoc.zip

# Install protoc plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
RUN go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
RUN go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest 
RUN go install github.com/envoyproxy/protoc-gen-validate@latest


FROM registry.access.redhat.com/ubi9/ubi-minimal:latest

COPY --from=builder /usr/local/bin/protoc /usr/local/bin/protoc
COPY --from=builder /go/bin/protoc-gen-go /usr/local/bin/protoc-gen-go
COPY --from=builder /go/bin/protoc-gen-go-grpc /usr/local/bin/protoc-gen-go-grpc
COPY --from=builder /go/bin/protoc-gen-go-http /usr/local/bin/protoc-gen-go-http
COPY --from=builder /go/bin/protoc-gen-openapi /usr/local/bin/protoc-gen-openapi
COPY --from=builder /go/bin/protoc-gen-validate /usr/local/bin/protoc-gen-validate