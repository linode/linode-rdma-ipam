package ipam

import (
	"errors"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFixture(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture %s: %v", name, err)
	}
}

func TestResolve_Success(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "05-rdma0.network", "[Network]\nAddress=192.0.2.18/24\n")

	got, err := Resolve("rdma0", dir)
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}

	want := netip.MustParsePrefix("192.0.2.18/24")
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestResolve_MissingFile(t *testing.T) {
	dir := t.TempDir()

	_, err := Resolve("rdma7", dir)
	if err == nil {
		t.Fatal("expected an error for a missing config file, got nil")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("expected error to wrap os.ErrNotExist, got: %v", err)
	}
	// The error should name both the interface and the path checked, so a
	// misconfigured deployment fails loudly and specifically.
	if !strings.Contains(err.Error(), "rdma7") || !strings.Contains(err.Error(), dir) {
		t.Errorf("error message %q should mention the interface name and directory checked", err.Error())
	}
}

func TestResolve_FileWithNoAddress(t *testing.T) {
	dir := t.TempDir()
	writeFixture(t, dir, "05-rdma3.network", "[Network]\nDHCP=no\n")

	_, err := Resolve("rdma3", dir)
	if err == nil {
		t.Fatal("expected an error, got nil")
	}
}

func TestResolve_EmptyIfName(t *testing.T) {
	dir := t.TempDir()

	_, err := Resolve("", dir)
	if err == nil {
		t.Fatal("expected an error for an empty interface name, got nil")
	}
}
