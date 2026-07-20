package hysteria2

import (
	"net/url"
	"testing"

	"github.com/mzz2017/gg/dialer"
	"gopkg.in/yaml.v3"
)

func TestNew(t *testing.T) {
	got, err := New("hysteria2://password@192.0.2.1:443?sni=proxy.example&insecure=1#hy2-node", &dialer.GlobalOption{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name() != "hy2-node" || got.Protocol() != "hysteria2" {
		t.Fatalf("metadata = (%q, %q)", got.Name(), got.Protocol())
	}
	if !got.SupportUDP() {
		t.Fatal("expected UDP support")
	}
	parsed, err := url.Parse(got.Link())
	if err != nil || parsed.Scheme != "hysteria2" {
		t.Fatalf("exported link = %q", got.Link())
	}
}

func TestNewFromClash(t *testing.T) {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte("name: clash-hy2\ntype: hysteria2\nserver: 192.0.2.1\nport: 443\npassword: secret\nsni: proxy.example\nskip-cert-verify: true\n"), &node); err != nil {
		t.Fatal(err)
	}
	got, err := NewFromClash(&node, &dialer.GlobalOption{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Name() != "clash-hy2" || !got.SupportUDP() {
		t.Fatalf("metadata = (%q, udp=%v)", got.Name(), got.SupportUDP())
	}
}

func TestNewFromClashWithoutPassword(t *testing.T) {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte("name: clash-hy2\ntype: hysteria2\nserver: 192.0.2.1\nport: 443\n"), &node); err != nil {
		t.Fatal(err)
	}
	got, err := NewFromClash(&node, &dialer.GlobalOption{})
	if err != nil {
		t.Fatal(err)
	}
	if parsed, err := url.Parse(got.Link()); err != nil {
		t.Fatal(err)
	} else if parsed.User != nil {
		t.Fatalf("unexpected empty user info in link %q", got.Link())
	}
}

func TestNewFromClashUDPDisabled(t *testing.T) {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte("name: clash-hy2\ntype: hysteria2\nserver: 192.0.2.1\nport: 443\npassword: secret\nudp: false\n"), &node); err != nil {
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
