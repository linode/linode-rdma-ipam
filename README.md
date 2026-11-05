# linode-rdma-ipam

A CNI IPAM plugin that resolves the correct static address for one of
Linode's RDMA rail interfaces (`rdma0`-`rdma7`) by reading the
`systemd-networkd` config file Linode's Network Helper already writes for
it on every boot, instead of drawing from an address pool.

## Development

```sh
mise check   # gofmt + go vet + go test ./...
mise build   # static Linux binary at bin/linode-rdma-ipam
mise cover   # test coverage report
```

Developed and tested against Go 1.26.5 on Linux. The core packages
(`pkg/networkd`, `pkg/ipam`) are portable Go with no Linux-only syscalls,
but the build and full test suite were run on Linux for parity with
where this plugin actually executes as a real CNI invocation.

## Deployment

1. Build the image: `mise ko-build`
2. Get the image onto both cluster nodes. For this small, bespoke 2-node
   cluster (no existing registry), build locally and `ctr image import` it
   directly onto each node rather than pushing to a registry - a
   registry-based push is the correct path once this scales beyond a
   handful of nodes, but adds a dependency (registry credentials/URL) not
   currently part of this setup.
3. If migrating from the previous manifest-based install, remove its
   unmanaged `DaemonSet` first so Helm can create and manage it:
   `kubectl delete daemonset linode-rdma-ipam-install -n nvidia-network-operator`.
4. Run `mise run helm-install` to install or upgrade the chart in
   `nvidia-network-operator`. It copies the binary into `/opt/cni/bin` on
   every matching node (the same pattern as the `cni-plugins-ds`/
   `kube-multus-ds` DaemonSets already running in this cluster) and self-heals
   on node reboot or replacement.
5. Confirm `/opt/cni/bin/linode-rdma-ipam` exists on both nodes.
