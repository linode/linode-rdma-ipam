# top-level justfile for linode-rdma-ipam
## Image tags to publish; defaults to the current branch name when unset
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

# Remove generated binaries and coverage output.
clean:
    rm -rf {{quote(LOCALBIN)}}
    rm -rf bin coverage.out
