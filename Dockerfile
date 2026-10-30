# Multi-stage build: compile a static binary, ship it in a minimal image.
# The final image is never actually run as a workload - see
# manifests/linode-rdma-ipam-install.yaml, which uses an initContainer's
# `cp` to copy the binary out onto the host's /opt/cni/bin. That's why the
# final stage is busybox rather than scratch: we need a `cp` binary in
# there to do the copy, since the payload itself has no shell of its own.

FROM --platform=$BUILDPLATFORM golang:1.26 AS builder
ARG TARGETOS=linux
ARG TARGETARCH=amd64
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -o /out/linode-rdma-ipam .

FROM --platform=$TARGETPLATFORM busybox:stable
COPY --from=builder /out/linode-rdma-ipam /linode-rdma-ipam
