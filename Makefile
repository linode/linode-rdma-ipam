#####################################################################
# top-level Makefile for linode-rdma-ipam
#####################################################################
REGISTRY            ?= docker.io/linode
IMAGE_NAME          ?= linode-rdma-ipam
VERSION             ?= $(shell git describe --always --tag --dirty=-dev)
IMAGE_TAGS          ?= $(VERSION)
KO_DOCKER_REPO      ?= docker.io/linode/linode-rdma-ipam
KOCACHE             ?= $(HOME)/.ko
# ko's --local path uses DOCKER_HOST directly rather than Docker CLI contexts.
DOCKER_HOST          ?= $(shell docker context inspect --format '{{.Endpoints.docker.Host}}')
BUILD_ARGS          := --build-arg VERSION=$(VERSION)
SHELL                = /usr/bin/env bash -o pipefail
.SHELLFLAGS          = -ec

#####################################################################
# OS / ARCH
#####################################################################
OS=$(shell uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(shell uname -m)
ARCH_SHORT=$(ARCH)
ifeq ($(ARCH_SHORT),x86_64)
ARCH_SHORT := amd64
else ifeq ($(ARCH_SHORT),aarch64)
ARCH_SHORT := arm64
endif
# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

#####################################################################
##@ Build All
#####################################################################
.PHONY: all
all: check build

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk command is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php


## --------------------------------------
## Help
## --------------------------------------

##@ Help:

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## --------------------------------------
## Build
## --------------------------------------

.PHONY: build
build: fmt vet ## Build manager binary.
	go build -o ./bin/linode-rdma-ipam .

.PHONY: ko-build
ko-build:
	DOCKER_HOST=$(DOCKER_HOST) KOCACHE=$(KOCACHE) ko build --local -t $(IMAGE_TAGS) --bare github.com/linode/linode-rdma-ipam

.PHONY: ko-publish
ko-publish:
	KOCACHE=$(KOCACHE) KO_DOCKER_REPO=$(KO_DOCKER_REPO) ko build -t $(IMAGE_TAGS) --bare github.com/linode/linode-rdma-ipam

## --------------------------------------
## Testing
## --------------------------------------

##@ Testing:

## Run the full test suite (pkg/networkd, pkg/ipam, and the top-level
## cmdAdd/cmdCheck/cmdDel wiring tests via testutils.Cmd*WithArgs).
test:
	go test ./... -count=1 -race -timeout 60s -v

## Run tests with coverage output.
cover:
	go test ./... -count=1 -race -timeout 60s -coverprofile=coverage.out
	go tool cover -func=coverage.out

## --------------------------------------
## Development
## --------------------------------------

##@ Development:

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: tidy
tidy: ## Run go mod tidy against code.
	go mod tidy

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: lint
lint: ## Run lint against code.
	golangci-lint run -c .golangci.yml

.PHONY: nilcheck
nilcheck: golangci-lint-custom ## Run nil check against code.
	$(GOLANGCI_LINT_CUSTOM) run -c .golangci-nilaway.yml

.PHONY: vulncheck
vulncheck: ## Run vulnerability check against code.
	govulncheck ./...

## --------------------------------------
## Deployment
## --------------------------------------

##@ Deployment:

## install/upgrade the Helm chart using the current image (IMG).
.PHONY: helm-install
HELM_NAMESPACE ?= nvidia-network-operator
helm-install:
	helm upgrade --install linode-rdma-ipam charts/linode-rdma-ipam \
		--namespace $(HELM_NAMESPACE) \
		--create-namespace \
		--set image.repository=$(KO_DOCKER_REPO) \
		--set image.tag=$(VERSION)

clean:
	rm -rf $(LOCALBIN)
	rm -rf bin coverage.out

## --------------------------------------------
## Build deps (that can't be installed by mise)
## --------------------------------------------

CACHE_BIN ?= $(CURDIR)/bin
LOCALBIN ?= $(CACHE_BIN)

NILAWAY              ?= $(LOCALBIN)/nilaway
GOLANGCI_LINT_CUSTOM ?= $(CACHE_BIN)/golangci-lint-custom

# renovate: datasource=go depName=go.uber.org/nilaway
NILAWAY_VERSION          ?= d2274102dc2eab9f77cef849a5470a6ebf983125

.PHONY: nilaway
nilaway: $(NILAWAY) ## Download nilaway locally if necessary.

$(NILAWAY): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install go.uber.org/nilaway/cmd/nilaway@$(NILAWAY_VERSION)

.PHONY: golangci-lint-custom
golangci-lint-custom: $(GOLANGCI_LINT_CUSTOM)
$(GOLANGCI_LINT_CUSTOM): # Build golangci-lint-custom from custom configuration.
	golangci-lint custom --version v$(shell mise current golangci-lint)

