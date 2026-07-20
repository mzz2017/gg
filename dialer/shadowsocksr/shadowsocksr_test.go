package shadowsocksr

import (
	"context"
	"errors"
	"testing"

	"github.com/daeuniverse/outbound/netproxy"
)

type netproxyDialerFunc func(context.Context, string, string) (netproxy.Conn, error)

func (f netproxyDialerFunc) DialContext(ctx context.Context, network, addr string) (netproxy.Conn, error) {
	return f(ctx, network, addr)
}

func TestOutboundDialerUsesSSRServer(t *testing.T) {
	wantErr := errors.New("stop before connecting")
	var gotNetwork, gotAddress string
	next := netproxyDialerFunc(func(_ context.Context, network, addr string) (netproxy.Conn, error) {
		gotNetwork = network
		gotAddress = addr
		return nil, wantErr
	})

	s := &ShadowsocksR{
		Name:       "test proxy",
		Server:     "proxy.example",
		Port:       8388,
		Password:   "password",
		Cipher:     "aes-256-cfb",
		Proto:      "auth_chain_a",
		ProtoParam: "protocol parameter",
		Obfs:       "tls1.2_ticket_auth",
		ObfsParam:  "obfs.example",
		Protocol:   "shadowsocksr",
	}
	d, err := s.dialer(next)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Dial("tcp", "target.example:443"); !errors.Is(err, wantErr) {
		t.Fatalf("dial error = %v, want %v", err, wantErr)
	}
	if gotNetwork != "tcp" || gotAddress != "proxy.example:8388" {
		t.Fatalf("next dial = %q %q, want tcp proxy.example:8388", gotNetwork, gotAddress)
	}
	if d.Name() != s.Name || d.Protocol() != s.Protocol || d.Link() != s.ExportToURL() || d.SupportUDP() {
		t.Fatalf("dialer metadata = name %q, protocol %q, link %q, udp %t", d.Name(), d.Protocol(), d.Link(), d.SupportUDP())
	}
}

func TestOutboundDialerRejectsUnsupportedObfs(t *testing.T) {
	s := &ShadowsocksR{
		Server:   "proxy.example",
		Port:     8388,
		Password: "password",
		Cipher:   "aes-256-cfb",
		Proto:    "origin",
		Obfs:     "unsupported",
		Protocol: "shadowsocksr",
	}
	next := netproxyDialerFunc(func(context.Context, string, string) (netproxy.Conn, error) {
		return nil, errors.New("unexpected dial")
	})
	if _, err := s.dialer(next); err == nil {
		t.Fatal("unsupported obfs was accepted")
	}
}
