package anytls

import (
	"net/url"
	"testing"

	"github.com/mzz2017/gg/dialer"
	"gopkg.in/yaml.v3"
)

func TestNew(t *testing.T) {
	got, err := New(
		"anytls://test-auth@192.0.2.1:443?sni=proxy.example&insecure=1#anytls-node",
		&dialer.GlobalOption{},
	)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name() != "anytls-node" {
		t.Fatalf("name = %q, want anytls-node", got.Name())
	}
	if got.Protocol() != "anytls" {
		t.Fatalf("protocol = %q, want anytls", got.Protocol())
	}
	if !got.SupportUDP() {
		t.Fatal("expected UDP support")
	}
	link, err := url.Parse(got.Link())
	if err != nil {
		t.Fatal(err)
	}
	if link.Scheme != "anytls" {
		t.Fatalf("link scheme = %q, want anytls", link.Scheme)
	}
}

func TestNewFromClash(t *testing.T) {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte("name: clash-anytls\ntype: anytls\nserver: 192.0.2.1\nport: 443\npassword: secret\nsni: proxy.example\nskip-cert-verify: true\n"), &node); err != nil {
		t.Fatal(err)
	}
	got, err := NewFromClash(&node, &dialer.GlobalOption{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name() != "clash-anytls" || got.Protocol() != "anytls" || !got.SupportUDP() {
		t.Fatalf("metadata = (%q, %q, udp=%v)", got.Name(), got.Protocol(), got.SupportUDP())
	}
}

func TestNewFromClashUDPDisabled(t *testing.T) {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte("name: clash-anytls\ntype: anytls\nserver: 192.0.2.1\nport: 443\npassword: secret\nudp: false\n"), &node); err != nil {
		t.Fatal(err)
	}
	got, err := NewFromClash(&node, &dialer.GlobalOption{})
	if err != nil {
		t.Fatal(err)
	}
	if got.SupportUDP() {
		t.Fatal("expected UDP support to be disabled")
	}
}
