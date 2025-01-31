FROM registry.access.redhat.com/ubi8/ubi-minimal:8.10-1179 AS builder

ARG TARGETARCH
USER root
RUN microdnf install -y tar gzip make which go-toolset

WORKDIR /workspace

COPY . ./

ENV CGO_ENABLED 1
RUN go mod vendor
RUN make build

# adds fips-detect tool for FIPS validation -- likely not needed long term
RUN mkdir /tmp/go && GOPATH=/tmp/go GOCACHE=/tmp/go go install github.com/acardace/fips-detect@latest

FROM registry.access.redhat.com/ubi8/ubi-minimal:8.10-1179

# installs RHEL fork of go to be able to validate with go tools for FIPS -- likely not needed long term
RUN microdnf install -y go-toolset
RUN mkdir /config

COPY --from=builder /workspace/bin/kessel-relations /usr/local/bin/
COPY --from=builder /tmp/go/bin/fips-detect /usr/local/bin/
COPY --from=builder /workspace/configs/config.yaml /config

EXPOSE 8000
EXPOSE 9000

USER 1001

ENTRYPOINT ["/usr/local/bin/kessel-relations","-conf","/config/config.yaml"]

LABEL name="kessel-relations-api" \
      version="0.0.1" \
      summary="Kessel relations-api service" \
      description="The Kessel relations-api service"
