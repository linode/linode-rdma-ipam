package networkd

import (
	"errors"
	"net/netip"
	"os"
	"strings"
	"testing"
)

func TestParseAddress_RealFixture(t *testing.T) {
	t.Parallel()

	fixtureFile, err := os.Open("../../testdata/05-rdma0.network")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer fixtureFile.Close()

	got, err := ParseAddress(fixtureFile)
	if err != nil {
		t.Fatalf("ParseAddress: %v", err)
	}

	want := netip.MustParsePrefix("192.0.2.18/24")
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestParseAddress_NoAddressFixture(t *testing.T) {
	t.Parallel()

	fixtureFile, err := os.Open("../../testdata/05-rdma-no-address.network")
	if err != nil {
		t.Fatalf("open fixture: %v", err)
	}
	defer fixtureFile.Close()

	_, err = ParseAddress(fixtureFile)
	if !errors.Is(err, ErrNoAddress) {
		t.Fatalf("got err %v, want ErrNoAddress", err)
	}
}

func TestParseAddress_TableDriven(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name    string
		content string
		want    netip.Prefix
		wantErr error
	}{
		{
			name: "simple",
			content: "[Match]\n" +
				"Name=rdma0\n" +
				"\n" +
				"[Network]\n" +
				"Address=192.0.2.18/24\n",
			want: netip.MustParsePrefix("192.0.2.18/24"),
		},
		{
			name: "whitespace around equals",
			content: "[Network]\n" +
				"Address = 192.0.2.42/24 \n",
			want: netip.MustParsePrefix("192.0.2.42/24"),
		},
		{
			name: "crlf line endings",
			content: "[Network]\r\n" +
				"Address=192.0.2.52/24\r\n",
			want: netip.MustParsePrefix("192.0.2.52/24"),
		},
		{
			name: "whole-line comments and blank lines are ignored",
			content: "# a top of file comment\n" +
				"\n" +
				"[Network]\n" +
				"# another comment\n" +
				"\n" +
				"Address=192.0.2.2/24\n",
			want: netip.MustParsePrefix("192.0.2.2/24"),
		},
		{
			name: "first IPv4 Address line wins when there are multiple",
			content: "[Network]\n" +
				"Address=192.0.2.22/24\n" +
				"Address=192.0.2.99/24\n",
			want: netip.MustParsePrefix("192.0.2.22/24"),
		},
		{
			name: "IPv6-only Address lines are ignored, IPv4 still found",
			content: "[Network]\n" +
				"Address=2001:db8::1/64\n" +
				"Address=192.0.2.32/24\n",
			want: netip.MustParsePrefix("192.0.2.32/24"),
		},
		{
			name: "Address outside the Network section is ignored",
			content: "[Match]\n" +
				"Address=192.0.2.9/24\n" +
				"\n" +
				"[Network]\n" +
				"Address=192.0.2.62/24\n",
			want: netip.MustParsePrefix("192.0.2.62/24"),
		},
		{
			name: "no Network section at all",
			content: "[Match]\n" +
				"Name=rdma0\n",
			wantErr: ErrNoAddress,
		},
		{
			name: "Network section with only IPv6",
			content: "[Network]\n" +
				"Address=2001:db8::1/64\n",
			wantErr: ErrNoAddress,
		},
		{
			name: "malformed CIDR is skipped, not fatal, if a valid one follows",
			content: "[Network]\n" +
				"Address=not-an-address\n" +
				"Address=192.0.2.72/24\n",
			want: netip.MustParsePrefix("192.0.2.72/24"),
		},
		{
			name: "malformed CIDR with nothing valid after it",
			content: "[Network]\n" +
				"Address=not-an-address\n",
			wantErr: ErrNoAddress,
		},
		{
			name:    "empty file",
			content: "",
			wantErr: ErrNoAddress,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := ParseAddress(strings.NewReader(tc.content))
			if tc.wantErr != nil {
				if !errors.Is(err, tc.wantErr) {
					t.Fatalf("got err %v, want %v", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("ParseAddress: %v", err)
			}
			if got != tc.want {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}
