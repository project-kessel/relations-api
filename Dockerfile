FROM registry.access.redhat.com/ubi8/ubi-minimal:8.10-1130 AS builder

ARG TARGETARCH
USER root
RUN microdnf install -y tar gzip make which

# install platform specific go version
RUN curl -O -J  https://dl.google.com/go/go1.22.0.linux-${TARGETARCH}.tar.gz
RUN tar -C /usr/local -xzf go1.22.0.linux-${TARGETARCH}.tar.gz
RUN ln -s /usr/local/go/bin/go /usr/local/bin/go

WORKDIR /workspace

COPY . ./

RUN go mod vendor
RUN make build

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.10-1130

RUN mkdir /config
COPY --from=builder /workspace/bin/kessel-relations /usr/local/bin/
COPY --from=builder /workspace/configs/config.yaml /config

EXPOSE 8000
EXPOSE 9000

USER 1001

ENTRYPOINT ["/usr/local/bin/kessel-relations","-conf","/config/config.yaml"]

LABEL name="kessel-relations-api" \
      version="0.0.1" \
      summary="Kessel relations-api service" \
      description="The Kessel relations-api service"
