FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7-1770267347 AS builder

ARG TARGETARCH
USER root
RUN microdnf install -y tar gzip make which go-toolset

WORKDIR /workspace

COPY . ./

ENV CGO_ENABLED 1
RUN go mod vendor
RUN make build

FROM registry.access.redhat.com/ubi9/ubi-minimal:9.7-1770267347

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
