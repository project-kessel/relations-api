FIPS_ENABLED?=true

GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
GOBIN?=$(shell go env GOBIN)
GOFLAGS_MOD ?=
VERSION=$(shell git describe --tags --always)
DOCKER := $(shell type -P podman || type -P docker)

GOENV=GOOS=${GOOS} GOARCH=${GOARCH} CGO_ENABLED=1 GOFLAGS="${GOFLAGS_MOD}"
GOBUILDFLAGS=-gcflags="all=-trimpath=${GOPATH}" -asmflags="all=-trimpath=${GOPATH}"

ifeq ($(GOHOSTOS), windows)
	#the `find.exe` is different from `find` in bash/shell.
	#to see https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/find.
	#changed to use git-bash.exe to run find cli or other cli friendly, caused of every developer has a Git.
	#Git_Bash= $(subst cmd\,bin\bash.exe,$(dir $(shell where git)))
	Git_Bash=$(subst \,/,$(subst cmd\,bin\bash.exe,$(dir $(shell where git))))
endif

ifeq (${FIPS_ENABLED}, true)
GOFLAGS_MOD+=-tags=fips_enabled
GOFLAGS_MOD:=$(strip ${GOFLAGS_MOD})
GOENV+=GOEXPERIMENT=strictfipsruntime,boringcrypto
GOENV:=$(strip ${GOENV})
endif

.PHONY: init
# init env
init:
	go install github.com/go-kratos/kratos/cmd/kratos/v2@latest
	go install github.com/google/wire/cmd/wire@latest

.PHONY: config
# generate internal proto
config:
	@echo "Generating internal protos"
	@$(DOCKER) build -t custom-protoc ./api
	@$(DOCKER) run -t --rm -v $(PWD)/internal:/internal -v $(PWD)/third_party:/third_party \
	-w=/internal/conf/ custom-protoc sh -c "buf generate"

.PHONY: api
# generate api proto
api:
	@echo "Generating api protos"
	@$(DOCKER) build -t custom-protoc ./api
	@$(DOCKER) run -t --rm -v $(PWD)/api:/api -v $(PWD)/openapi.yaml:/openapi.yaml -v $(PWD)/third_party:/third_party \
	-w=/api/ custom-protoc sh -c "buf generate && \
		buf lint && \
		buf breaking --against 'buf.build/project-kessel/relations-api' "


.PHONY: build
# build
build:
	$(warning Setting GOEXPERIMENT=strictfipsruntime,boringcrypto - this generally causes builds to fail unless building inside the provided Dockerfile. If building locally, run `make local-build`)
	mkdir -p bin/ && ${GOENV} GOOS=${GOOS} go build ${GOBUILDFLAGS} -ldflags "-X cmd.Version=$(VERSION)" -o ./bin/ ./...

.PHONY: local-build
# local-build to ensure FIPS is not enabled which would likely result in a failed build locally
local-build:
	mkdir -p bin/ && go build -ldflags "-X cmd.Version=$(VERSION)" -o ./bin/ ./...

.PHONY: docker-build-push
docker-build-push:
	./build_deploy.sh

.PHONY: build-push-minimal
build-push-minimal:
	./build_push_minimal.sh

# run all tests
.PHONY: test
test:
	@echo ""
	@echo "Running tests."
	go test ./... -count=1

.PHONY: generate
# generate
generate:
	go mod tidy
	go get github.com/google/wire/cmd/wire@latest
	go generate ./...

.PHONY: all
# generate all
all:
	make api;
	make config;
	make generate;

# run go linter with the repositories lint config
.PHONY: lint
lint:
	@echo "Linting code."
	@$(DOCKER) run -t --rm -v $(PWD):/app -w /app golangci/golangci-lint golangci-lint run -v
	@$(DOCKER) run -t --rm -v $(PWD):/data pipelinecomponents/yamllint:latest \
		-c /data/.github/workflows/.yamllint /data/deploy/kessel-relations-deploy.yaml


.PHONY: pr-check
# generate pr-check
pr-check:
	make generate;
	make test;
	make lint;
	make local-build;

spicedb-up:
	./spicedb/start-spicedb.sh
.PHONY: spicedb-up

relations-api-up:
	./spicedb/start-relations-api.sh
.PHONY: relations-api-up

relations-api-down:
	./spicedb/stop-relations-api.sh
.PHONY: relations-api-down

spicedb-down:
	./spicedb/teardown.sh
.PHONY: spicedb-down

kind/relations-api:
	./spicedb-kind-setup/setup.sh
.PHONY: kind/relations-api

kind/teardown:
	./spicedb-kind-setup/teardown.sh
.PHONY: kind/teardown

.PHONY: run
# run api locally
run: local-build
	 ./bin/kessel-relations -conf configs

# show help
help:
	@echo ''
	@echo 'Usage:'
	@echo ' make [target]'
	@echo ''
	@echo 'Targets:'
	@awk '/^[a-zA-Z\-\_0-9]+:/ { \
	helpMessage = match(lastLine, /^# (.*)/); \
		if (helpMessage) { \
			helpCommand = substr($$1, 0, index($$1, ":")); \
			helpMessage = substr(lastLine, RSTART + 2, RLENGTH); \
			printf "\033[36m%-22s\033[0m %s\n", helpCommand,helpMessage; \
		} \
	} \
	{ lastLine = $$0 }' $(MAKEFILE_LIST)

.DEFAULT_GOAL := help
