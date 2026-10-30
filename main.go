// Command linode-rdma-ipam is a CNI IPAM plugin that resolves the correct
// static address for one of Linode's RDMA rail interfaces (rdma0-rdma7) by
// reading the systemd-networkd config file Linode's Network Helper already
// writes for it on every boot, instead of drawing from a pool.
//
// It relies on the calling main plugin (host-device, unmodified) preserving
// the interface's original host-side name across the move into the pod's
// network namespace, so CNI_IFNAME arrives as e.g. "rdma0" rather than a
// generic "net1" - see the repo root README.md for the full design
// rationale.
package main

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/containernetworking/cni/pkg/skel"
	"github.com/containernetworking/cni/pkg/types"
	types100 "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/cni/pkg/version"

	"linode-rdma-ipam/pkg/ipam"
)

// ipamArgs holds the fields this plugin reads from the "ipam" sub-object of
// the delegating main plugin's config - that's the whole netconf blob CNI
// passes an IPAM plugin as stdin, per the CNI IPAM delegation convention.
type ipamArgs struct {
	Type string `json:"type"`
	// NetworkDir overrides where Linode's Network Helper writes the
	// per-interface .network files. Defaults to ipam.DefaultNetworkdDir.
	NetworkDir string `json:"networkDir,omitempty"`
}

// NetConf is the CNI network configuration this plugin expects. It shadows
// types.NetConf's own minimal IPAM field with our richer ipamArgs so we can
// read networkDir alongside type.
type NetConf struct {
	types.NetConf
	IPAM ipamArgs `json:"ipam"`
}

func parseConf(stdin []byte) (*NetConf, error) {
	conf := &NetConf{}
	if err := json.Unmarshal(stdin, conf); err != nil {
		return nil, fmt.Errorf("failed to parse network configuration: %w", err)
	}
	if conf.IPAM.NetworkDir == "" {
		conf.IPAM.NetworkDir = ipam.DefaultNetworkdDir
	}
	return conf, nil
}

func cmdAdd(args *skel.CmdArgs) error {
	conf, err := parseConf(args.StdinData)
	if err != nil {
		return err
	}

	prefix, err := ipam.Resolve(args.IfName, conf.IPAM.NetworkDir)
	if err != nil {
		return err
	}

	result := &types100.Result{
		CNIVersion: conf.CNIVersion,
		IPs: []*types100.IPConfig{
			{
				Address: net.IPNet{
					IP:   net.IP(prefix.Addr().AsSlice()),
					Mask: net.CIDRMask(prefix.Bits(), prefix.Addr().BitLen()),
				},
				// No gateway: these are isolated rail subnets with no
				// cross-rail routing, so guessing one would be worse than
				// omitting it.
			},
		},
	}

	return types.PrintResult(result, conf.CNIVersion)
}

// cmdCheck re-derives the expected address for the interface and confirms
// it's still resolvable. It does not inspect live interface state (that
// would require netlink/root access, out of scope for this version).
func cmdCheck(args *skel.CmdArgs) error {
	conf, err := parseConf(args.StdinData)
	if err != nil {
		return err
	}

	_, err = ipam.Resolve(args.IfName, conf.IPAM.NetworkDir)
	return err
}

// cmdDel is a no-op: the address is always deterministically re-derived
// from the interface name rather than drawn from a pool, so there is no
// lease-file state to release.
func cmdDel(_ *skel.CmdArgs) error {
	return nil
}

func main() {
	skel.PluginMain(cmdAdd, cmdCheck, cmdDel, version.All, "linode-rdma-ipam")
}
