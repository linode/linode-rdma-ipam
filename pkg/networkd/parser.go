// Package networkd parses systemd-networkd .network files, specifically the
// [Network] Address= directive. Linode's "Network Helper" writes one such
// file per RDMA interface (e.g. /etc/systemd/network/05-rdma0.network) on
// every boot, containing the address that interface should have.
package networkd

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net/netip"
	"strings"
)

// ErrNoAddress is returned when no usable IPv4 Address= line was found in
// the [Network] section.
var ErrNoAddress = errors.New("no IPv4 Address= found in [Network] section")

// ParseAddress reads a systemd-networkd .network file and returns the first
// IPv4 address configured via Address= in the [Network] section. IPv6
// addresses, other sections, comments, and blank lines are ignored. If
// multiple IPv4 Address= lines are present, the first one wins.
func ParseAddress(r io.Reader) (netip.Prefix, error) {
	scanner := bufio.NewScanner(r)

	section := ""
	for scanner.Scan() {
		line := strings.TrimSpace(strings.TrimRight(scanner.Text(), "\r"))
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			continue
		}

		if section != "Network" {
			continue
		}

		key, value, ok := splitDirective(line)
		if !ok || key != "Address" {
			continue
		}

		prefix, err := netip.ParsePrefix(value)
		if err != nil {
			// Not a fatal error: keep scanning in case a later, valid
			// Address= line follows.
			continue
		}
		if !prefix.Addr().Is4() {
			continue
		}
		return prefix, nil
	}

	if err := scanner.Err(); err != nil {
		return netip.Prefix{}, fmt.Errorf("reading network config: %w", err)
	}

	return netip.Prefix{}, ErrNoAddress
}

// splitDirective splits a "Key=Value" line, trimming whitespace around the
// key and value. Returns ok=false if there's no '=' in the line.
func splitDirective(line string) (key, value string, ok bool) {
	idx := strings.IndexByte(line, '=')
	if idx < 0 {
		return "", "", false
	}
	key = strings.TrimSpace(line[:idx])
	value = strings.TrimSpace(line[idx+1:])
	return key, value, true
}
