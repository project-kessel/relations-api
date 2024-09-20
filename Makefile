GOHOSTOS:=$(shell go env GOHOSTOS)
GOPATH:=$(shell go env GOPATH)
VERSION=$(shell git describe --tags --always)
DOCKER := $(shell type -P podman || type -P docker)

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
		

.PHONY: build
# build
build:
	mkdir -p bin/ && go build -ldflags "-X main.Version=$(VERSION)" -o ./bin/ ./...

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
