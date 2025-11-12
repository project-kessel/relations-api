FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7-1762956380 AS builder

ARG TARGETARCH
USER root
RUN microdnf install -y tar gzip unzip go-toolset

# Install protoc plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
RUN go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest

# Install Buf
RUN go install github.com/bufbuild/buf/cmd/buf@v1.52.0

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7-1762956380


COPY --from=builder /root/go/bin/protoc-gen-go /usr/local/bin/protoc-gen-go
COPY --from=builder /root/go/bin/protoc-gen-go-grpc /usr/local/bin/protoc-gen-go-grpc
COPY --from=builder /root/go/bin/protoc-gen-go-http /usr/local/bin/protoc-gen-go-http
COPY --from=builder /root/go/bin/buf /usr/local/bin/buf
