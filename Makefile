BINARY           := linode-rdma-ipam
GOOS             ?= linux
GOARCH           ?= amd64
USER             ?= $(shell whoami)
REGISTRY         ?= docker.io
REGISTRY_ORG     ?= ${USER}
IMAGE_NAME       ?= linode-rdma-ipam
PLUGIN_IMAGE     ?= $(REGISTRY)/$(REGISTRY_ORG)/$(IMAGE_NAME)
GIT_REF          ?= $(shell git rev-parse --short HEAD)
VERSION          ?= $(shell git describe --always --tag --dirty=-dev)
GITCOMMITSHA     := $(shell git rev-parse --verify HEAD)
BUILDTIMESTAMP   ?= $(shell date -u +"%FT%T%z")
TAG              ?= v0.0.0-$(GIT_REF)-$(USER)-dev
IMG              ?= ${PLUGIN_IMAGE}:${TAG}
PLATFORM         ?= linux/amd64



.PHONY: all build test cover fmt vet tidy check docker-build docker-push helm-install helm-upgrade clean

all: check build

## Build the CNI plugin binary for deployment (static, Linux target by default).
build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o bin/$(BINARY) .

## Run the full test suite (pkg/networkd, pkg/ipam, and the top-level
## cmdAdd/cmdCheck/cmdDel wiring tests via testutils.Cmd*WithArgs).
test:
	go test ./... -count=1 -v

## Run tests with coverage output.
cover:
	go test ./... -count=1 -coverprofile=coverage.out
	go tool cover -func=coverage.out

fmt:
	gofmt -l -w .

vet:
	go vet ./...

tidy:
	go mod tidy

## fmt + vet + test, meant to be run before committing or building.
check: fmt vet test

## Build the deployment container image (see Dockerfile).
docker-build:
	docker build --platform=$(PLATFORM) -t $(IMG) .

docker-push: docker-build
	docker push $(IMG)

## install/upgrade the Helm chart using the current image (IMG).
helm-install:
	helm upgrade --install linode-rdma-ipam charts/linode-rdma-ipam \
		--set image.repository=$(PLUGIN_IMAGE) \
		--set image.tag=$(TAG)

clean:
	rm -rf bin coverage.out
