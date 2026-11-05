# top-level justfile for linode-rdma-ipam
## Image tags to publish; defaults to the current branch name when unset
IMAGE_TAGS := env("IMAGE_TAGS", "")
IMAGE_VERSION := env("IMAGE_VERSION", "")
KO_DOCKER_REPO := env_var_or_default("KO_DOCKER_REPO", "docker.io/linode/linode-rdma-ipam")
KOCACHE := env_var_or_default("KOCACHE", home_directory() + "/.ko")
DEFAULT_DOCKER_HOST := `docker context inspect --format '{{.Endpoints.docker.Host}}'`
DOCKER_HOST := env_var_or_default("DOCKER_HOST", DEFAULT_DOCKER_HOST)
HELM_NAMESPACE := env_var_or_default("HELM_NAMESPACE", "nvidia-network-operator")
CACHE_BIN := env_var_or_default("CACHE_BIN", justfile_directory() + "/bin")
LOCALBIN := env_var_or_default("LOCALBIN", CACHE_BIN)
NILAWAY := env_var_or_default("NILAWAY", LOCALBIN + "/nilaway")
GOLANGCI_LINT_CUSTOM := env_var_or_default("GOLANGCI_LINT_CUSTOM", CACHE_BIN + "/golangci-lint-custom")
NILAWAY_VERSION := env_var_or_default("NILAWAY_VERSION", "d2274102dc2eab9f77cef849a5470a6ebf983125")

# Run checks and build the binary.
default: help
all: check build

# Display the available recipes.
help:
    @just --list

# Run formatting, vetting, and linting.
check: fmt vet lint

# Build the manager binary.
build: fmt vet
    go build -o ./bin/linode-rdma-ipam .

# Build the image locally with ko.
ko-build:
    #!/usr/bin/env bash
    set -euo pipefail
    tags="{{ IMAGE_TAGS }}"
    if [ -z "$tags" ]; then tags=$(git rev-parse --abbrev-ref HEAD); fi
    DOCKER_HOST={{quote(DOCKER_HOST)}} KOCACHE={{quote(KOCACHE)}} ko build --local -t "$tags" --bare github.com/linode/linode-rdma-ipam

# Build and push the image with ko.
ko-publish:
    #!/usr/bin/env bash
    set -euo pipefail
    tags="{{ IMAGE_TAGS }}"
    if [ -z "$tags" ]; then tags=$(git rev-parse --abbrev-ref HEAD); fi
    KOCACHE={{quote(KOCACHE)}} KO_DOCKER_REPO={{quote(KO_DOCKER_REPO)}} ko build -t "$tags" --bare github.com/linode/linode-rdma-ipam

# Run the full test suite.
test:
    go test ./... -count=1 -race -timeout 60s  -cover -coverprofile=coverage.out -outputdir=. -coverpkg=./...

# Run tests with coverage output.
cover: test
	go tool cover -html coverage.out -o coverage.html

# Format Go sources.
fmt:
    go fmt ./...

# Run go mod tidy.
tidy:
    go mod tidy

# Run go vet.
vet:
    go vet ./...

# Run golangci-lint.
lint:
    golangci-lint run -c .golangci.yml

# Run govulncheck.
vulncheck:
    govulncheck ./...

# Run nilaway.
nilcheck: golangci-lint-custom
    {{quote(GOLANGCI_LINT_CUSTOM)}} run -c .golangci-nilaway.yml

# Install nilaway locally if it is missing.
nilaway:
    if [ ! -f {{quote(NILAWAY)}} ]; then mkdir -p {{quote(LOCALBIN)}} && GOBIN={{quote(LOCALBIN)}} go install go.uber.org/nilaway/cmd/nilaway@{{quote(NILAWAY_VERSION)}}; fi

# Build golangci-lint-custom if it is missing.
golangci-lint-custom:
    if [ ! -f {{quote(GOLANGCI_LINT_CUSTOM)}} ]; then golangci-lint custom --version "v$(mise current golangci-lint)"; fi

lint-chart target_branch='main':
	ct lint \
	--chart-yaml-schema=.ct/chart_schema.yaml \
	--lint-conf=.ct/lintconf.yaml \
	--check-version-increment=false \
	--target-branch={{ target_branch }}

# Install or upgrade the Helm chart.
helm-install:
    helm upgrade --install linode-rdma-ipam charts/linode-rdma-ipam \
        --namespace {{quote(HELM_NAMESPACE)}} \
        --create-namespace \
        --set image.repository={{quote(KO_DOCKER_REPO)}} \
        --set image.tag={{quote(IMAGE_VERSION)}}

# Stamp a version into the chart metadata and controller image tag
set-chart-version version=IMAGE_VERSION:
	#!/usr/bin/env bash
	set -euo pipefail
	if [ -z "{{ version }}" ]; then
		echo "IMAGE_VERSION is required, e.g. just set-chart-version v0.1.6" >&2
		exit 1
	fi
	for chart in ./charts/*/Chart.yaml; do
		IMAGE_VERSION="{{ version }}" yq -i '.version = strenv(IMAGE_VERSION) | .appVersion = strenv(IMAGE_VERSION)' "$chart"
	done
	IMAGE_VERSION="{{ version }}" yq -i '.image.tag = strenv(IMAGE_VERSION)' ./charts/linode-rdma-ipam/values.yaml

# Package the versioned Helm chart for a tagged release.
release:
    #!/usr/bin/env bash
    set -euo pipefail
    : "${IMAGE_VERSION:?IMAGE_VERSION must be set}"
    chart_version="${IMAGE_VERSION#v}"
    mkdir -p release
    helm package charts/linode-rdma-ipam \
        --destination release \
        --version "$chart_version" \
        --app-version "$IMAGE_VERSION"
    mv "release/linode-rdma-ipam-${chart_version}.tgz" \
        "release/helm-chart-${IMAGE_VERSION}.tgz"

# Remove generated binaries and coverage output.
clean:
    rm -rf {{quote(LOCALBIN)}}
    rm -rf bin coverage.out
