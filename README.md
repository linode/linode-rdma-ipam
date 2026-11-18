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

## Usage

The release binary will be present on LKE nodes under `/opt/cni/bin/linode-rdma-ipam`. LKE will use this CNI plugin as needed.


## 🚧 Disclaimers

- **The CNI plugin is in alpha.** Behavior may change between releases.

## 💬 Join Us on Slack

- **General Help/Discussion**: [Kubernetes Slack - #linode](https://kubernetes.slack.com/messages/CD4B15LUR)

## License

This project is licensed under the Apache License, Version 2.0.
