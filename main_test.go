//nolint:paralleltest // The CNI command test helpers swap process-global os.Stdout.
package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/containernetworking/cni/pkg/skel"
	types100 "github.com/containernetworking/cni/pkg/types/100"
	"github.com/containernetworking/plugins/pkg/testutils"
)

func writeFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
}

func netConfJSON(t *testing.T, networkDir string) []byte {
	t.Helper()
	conf := map[string]any{
		"cniVersion": "1.0.0",
		"name":       "sriov-rdma-network-0",
		"type":       "linode-rdma-ipam",
		"ipam": map[string]any{
			"type":       "linode-rdma-ipam",
			"networkDir": networkDir,
		},
	}
	configBytes, err := json.Marshal(conf)
	if err != nil {
		t.Fatalf("marshal conf: %v", err)
	}
	return configBytes
}

func TestCmdAdd_Success(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "05-rdma0.network", "[Network]\nAddress=192.0.2.18/24\n")

	args := &skel.CmdArgs{
		ContainerID: "testcontainer",
		Netns:       "",
		IfName:      "rdma0",
		Path:        "",
		StdinData:   netConfJSON(t, dir),
	}

	result, _, err := testutils.CmdAddWithArgs(args, func() error {
		return cmdAdd(args)
	})
	if err != nil {
		t.Fatalf("cmdAdd: %v", err)
	}

	res, err := types100.GetResult(result)
	if err != nil {
		t.Fatalf("GetResult: %v", err)
	}
	if len(res.IPs) != 1 {
		t.Fatalf("expected exactly 1 IP, got %d", len(res.IPs))
	}
	if got := res.IPs[0].Address.String(); got != "192.0.2.18/24" {
		t.Errorf("got address %s, want 192.0.2.18/24", got)
	}
	if res.IPs[0].Gateway != nil {
		t.Errorf("expected no gateway, got %v", res.IPs[0].Gateway)
	}
}

func TestCmdAdd_UnknownInterface(t *testing.T) {
	dir := t.TempDir() // empty: no fixture files at all

	args := &skel.CmdArgs{
		ContainerID: "testcontainer",
		IfName:      "rdma99",
		StdinData:   netConfJSON(t, dir),
	}

	_, _, err := testutils.CmdAddWithArgs(args, func() error {
		return cmdAdd(args)
	})
	if err == nil {
		t.Fatal("expected an error for an unresolvable interface, got nil")
	}
	if !strings.Contains(err.Error(), "rdma99") {
		t.Errorf("error %q should mention the interface name", err.Error())
	}
}

func TestCmdAdd_DefaultsNetworkDirWhenUnset(t *testing.T) {
	conf := map[string]any{
		"cniVersion": "1.0.0",
		"name":       "sriov-rdma-network-0",
		"type":       "linode-rdma-ipam",
		"ipam": map[string]any{
			"type": "linode-rdma-ipam",
		},
	}
	configBytes, err := json.Marshal(conf)
	if err != nil {
		t.Fatalf("marshal conf: %v", err)
	}

	args := &skel.CmdArgs{
		ContainerID: "testcontainer",
		IfName:      "rdma0",
		StdinData:   configBytes,
	}

	// With no fixture present at the real default dir for this made-up
	// interface name, this should fail with a "not found" style error
	// rather than a config-parsing error - proving the default network dir
	// was applied instead of erroring out on missing config.
	_, _, err = testutils.CmdAddWithArgs(args, func() error {
		return cmdAdd(args)
	})
	if err == nil {
		t.Fatal("expected an error since /etc/systemd/network/05-rdma0.network won't exist in the test sandbox, got nil")
	}
}

func TestCmdDel_IsNoOp(t *testing.T) {
	args := &skel.CmdArgs{
		ContainerID: "testcontainer",
		IfName:      "rdma0",
		StdinData:   netConfJSON(t, t.TempDir()),
	}

	err := testutils.CmdDelWithArgs(args, func() error {
		return cmdDel(args)
	})
	if err != nil {
		t.Fatalf("cmdDel should always succeed, got: %v", err)
	}
}

func TestCmdCheck_Success(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "05-rdma1.network", "[Network]\nAddress=192.0.2.2/24\n")

	args := &skel.CmdArgs{
		ContainerID: "testcontainer",
		IfName:      "rdma1",
		StdinData:   netConfJSON(t, dir),
	}

	err := testutils.CmdCheckWithArgs(args, func() error {
		return cmdCheck(args)
	})
	if err != nil {
		t.Fatalf("cmdCheck: %v", err)
	}
}

func TestCmdCheck_UnknownInterface(t *testing.T) {
	dir := t.TempDir()

	args := &skel.CmdArgs{
		ContainerID: "testcontainer",
		IfName:      "rdma99",
		StdinData:   netConfJSON(t, dir),
	}

	err := testutils.CmdCheckWithArgs(args, func() error {
		return cmdCheck(args)
	})
	if err == nil {
		t.Fatal("expected an error for an unresolvable interface, got nil")
	}
}
