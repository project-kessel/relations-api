IMAGE_TAG := $(shell git rev-parse --short=7 HEAD)
GIT_COMMIT := $(shell git rev-parse --short HEAD)

ifeq ($(DOCKER),)
DOCKER := $(shell command -v podman || command -v docker)
endif

ifeq ($(VERSION),)
VERSION := $(shell git describe --tags --always)
endif

# On macOS (Apple Silicon) cross-compile to linux/amd64 to match the target platform
ifeq ($(shell uname -s),Darwin)
PLATFORM_FLAGS := --platform linux/amd64 --build-arg TARGETARCH=amd64
else
PLATFORM_FLAGS :=
endif

GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
GOOS?=$(shell go env GOOS)
GOARCH?=$(shell go env GOARCH)
GOBIN?=$(shell go env GOBIN)

GOENV=GOOS=${GOOS} GOARCH=${GOARCH}
GOBUILDFLAGS=-gcflags="all=-trimpath=${GOPATH}" -asmflags="all=-trimpath=${GOPATH}"

ifeq ($(GOHOSTOS), windows)
	#the `find.exe` is different from `find` in bash/shell.
	#to see https://docs.microsoft.com/en-us/windows-server/administration/windows-commands/find.
	#changed to use git-bash.exe to run find cli or other cli friendly, caused of every developer has a Git.
	#Git_Bash= $(subst cmd\,bin\bash.exe,$(dir $(shell where git)))
	Git_Bash=$(subst \,/,$(subst cmd\,bin\bash.exe,$(dir $(shell where git))))
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

.PHONY: api_breaking
# generate api proto
api_breaking:
	@echo "Generating api protos, allowing breaking changes"
	@$(DOCKER) build -t custom-protoc ./api
	@$(DOCKER) run -t --rm -v $(PWD)/api:/api:rw,z -v $(PWD)/openapi.yaml:/openapi.yaml:rw,z \
	-w=/api/ custom-protoc sh -c "buf dep update"
	@$(DOCKER) run -t --rm -v $(PWD)/api:/api:rw,z -v $(PWD)/openapi.yaml:/openapi.yaml:rw,z \
	-w=/api/ custom-protoc sh -c "buf generate && \
		buf lint"		


.PHONY: build
# build
build:
	mkdir -p bin/ && ${GOENV} go build ${GOBUILDFLAGS} -ldflags "-X cmd.Version=$(VERSION)" -o ./bin/ ./...

.PHONY: docker-build-push
docker-build-push: ## Build and push the container image; IMAGE and quay.io login are required
	@[ -n "$(DOCKER)" ] || { echo "Error: neither podman nor docker found. Please install one to continue."; exit 1; }
	@[ -n "$(IMAGE)" ] || { echo "IMAGE is required. Example: make docker-build-push IMAGE=quay.io/youruser/relations-api"; exit 1; }
	@printf '%s\n' "$(IMAGE)" | grep -qE '^[a-zA-Z0-9][a-zA-Z0-9._/:@-]*$$' || { echo "IMAGE contains invalid characters. Use format: quay.io/your-org/image-name"; exit 1; }
	@"$(DOCKER)" build $(PLATFORM_FLAGS) --build-arg GIT_COMMIT="$(GIT_COMMIT)" -t "$(IMAGE):$(IMAGE_TAG)" -f ./Dockerfile . || \
		{ echo "Build failed. If due to authentication, check your registry credentials and try again."; exit 1; }
	@"$(DOCKER)" push "$(IMAGE):$(IMAGE_TAG)" || \
		{ echo "Push failed. If due to authentication, run: $(DOCKER) login quay.io"; exit 1; }
	@"$(DOCKER)" tag "$(IMAGE):$(IMAGE_TAG)" "$(IMAGE):latest"
	@"$(DOCKER)" push "$(IMAGE):latest" || \
		{ echo "Push failed. If due to authentication, run: $(DOCKER) login quay.io"; exit 1; }

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
	make build;

spicedb-up:
	./spicedb/start-spicedb.sh
.PHONY: spicedb-up

# uses alternative postgres port to avoid conflicts
spicedb-alt-up:
	./spicedb/start-spicedb-alt-port.sh
.PHONY: spicedb-alt-up

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
run: build
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
