# Build stage -- the Go toolchain embeds the validated FIPS module in all binaries automatically.
FROM registry.access.redhat.com/hi/go:1.26.5-fips AS builder

WORKDIR /workspace

COPY . ./

RUN go mod vendor
RUN make build

# Runtime stage -- set GODEBUG so the binary runs in FIPS mode.
FROM registry.access.redhat.com/hi/core-runtime:2.42-openssl-fips

WORKDIR /

COPY --from=builder /workspace/bin/kessel-relations /usr/local/bin/
COPY --from=builder /workspace/configs/config.yaml /config/config.yaml

ENV GODEBUG=fips140=on

EXPOSE 8000
EXPOSE 9000

USER 1001

ENTRYPOINT ["/usr/local/bin/kessel-relations","-conf","/config/config.yaml"]

LABEL name="kessel-relations-api" \
      version="0.0.1" \
      summary="Kessel relations-api service" \
      description="The Kessel relations-api service"
