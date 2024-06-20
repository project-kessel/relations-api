FROM registry.access.redhat.com/ubi8/ubi-minimal:8.10 AS builder

ARG TARGETARCH
USER root
RUN microdnf install -y tar gzip unzip

# install platform specific go version
RUN curl -O -J  https://dl.google.com/go/go1.22.0.linux-${TARGETARCH}.tar.gz
RUN tar -C /usr/local -xzf go1.22.0.linux-${TARGETARCH}.tar.gz
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go


# Install protoc
RUN curl -L -o protoc.zip https://github.com/protocolbuffers/protobuf/releases/download/v27.1/protoc-27.1-linux-x86_64.zip \
    && unzip protoc.zip -d /usr/local/ && rm protoc.zip

# Install protoc plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
RUN go install github.com/go-kratos/kratos/cmd/protoc-gen-go-http/v2@latest
RUN go install github.com/google/gnostic/cmd/protoc-gen-openapi@latest 
RUN go install github.com/envoyproxy/protoc-gen-validate@latest

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.10

COPY --from=builder /usr/local/bin/protoc /usr/local/bin/protoc
COPY --from=builder /root/go/bin/protoc-gen-go /usr/local/bin/protoc-gen-go
COPY --from=builder /root/go/bin/protoc-gen-go-grpc /usr/local/bin/protoc-gen-go-grpc
COPY --from=builder /root/go/bin/protoc-gen-go-http /usr/local/bin/protoc-gen-go-http
COPY --from=builder /root/go/bin/protoc-gen-openapi /usr/local/bin/protoc-gen-openapi
COPY --from=builder /root/go/bin/protoc-gen-validate /usr/local/bin/protoc-gen-validate