// Package ipam resolves the correct static IP for one of Linode's RDMA rail
// interfaces (rdma0-rdma7) by reading the systemd-networkd config file
// Linode's Network Helper already writes for it on every boot.
package ipam

import (
	"errors"
	"fmt"
	"net/netip"
	"os"
	"path/filepath"

	"github.com/linode/linode-rdma-ipam/pkg/networkd"
)

// DefaultNetworkdDir is where Linode's Network Helper writes per-interface
// .network files.
const DefaultNetworkdDir = "/etc/systemd/network"

// Resolve returns the IPv4 address configured for ifName by reading
// <networkdDir>/05-<ifName>.network. ifName must be the interface's
// original host-side name (e.g. "rdma0") - the calling CNI plugin
// preserves that name across the move into the pod's network namespace
// instead of renaming to net1..net8 (see the repo root README.md).
func Resolve(ifName, networkdDir string) (netip.Prefix, error) {
	if ifName == "" {
		return netip.Prefix{}, errors.New("resolve: interface name must not be empty")
	}

	filename := fmt.Sprintf("05-%s.network", ifName)
	if !filepath.IsLocal(filename) || filepath.Base(filename) != filename {
		return netip.Prefix{}, fmt.Errorf("resolve: invalid interface name %q", ifName)
	}
	path := filepath.Join(networkdDir, filename)

	//nolint:gosec // G304: filename is a local component under the configured networkd directory.
	networkFile, err := os.Open(path)
	if err != nil {
		return netip.Prefix{}, fmt.Errorf("resolve: no network config found for interface %q at %s: %w", ifName, path, err)
	}

	prefix, parseErr := networkd.ParseAddress(networkFile)
	closeErr := networkFile.Close()
	if parseErr != nil {
		return netip.Prefix{}, fmt.Errorf("resolve: interface %q (%s): %w", ifName, path, errors.Join(parseErr, closeErr))
	}
	if closeErr != nil {
		return netip.Prefix{}, fmt.Errorf("resolve: close network config for interface %q (%s): %w", ifName, path, closeErr)
	}

	return prefix, nil
}
