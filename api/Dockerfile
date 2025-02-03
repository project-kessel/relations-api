FROM registry.access.redhat.com/ubi8/ubi-minimal:8.10-1179 AS builder

ARG TARGETARCH
USER root
RUN microdnf install -y tar gzip unzip

# install platform specific go version
RUN curl -O -J  https://dl.google.com/go/go1.22.7.linux-${TARGETARCH}.tar.gz
RUN tar -C /usr/local -xzf go1.22.7.linux-${TARGETARCH}.tar.gz
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go

# Install protoc plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
RUN go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest

# Install Buf
RUN go install github.com/bufbuild/buf/cmd/buf@latest

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.10-1179


COPY --from=builder /root/go/bin/protoc-gen-go /usr/local/bin/protoc-gen-go
COPY --from=builder /root/go/bin/protoc-gen-go-grpc /usr/local/bin/protoc-gen-go-grpc
COPY --from=builder /root/go/bin/protoc-gen-go-http /usr/local/bin/protoc-gen-go-http
COPY --from=builder /root/go/bin/buf /usr/local/bin/buf
