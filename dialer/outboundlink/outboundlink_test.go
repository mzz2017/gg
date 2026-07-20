package outboundlink

import (
	"context"
	"net/url"
	"testing"

	outbounddialer "github.com/daeuniverse/outbound/dialer"
	"github.com/daeuniverse/outbound/netproxy"
	"github.com/mzz2017/gg/dialer"
	"gopkg.in/yaml.v3"
)

type stubDialer struct{}

func (stubDialer) DialContext(context.Context, string, string) (netproxy.Conn, error) {
	return nil, nil
}

func constructor(_ *outbounddialer.ExtraOption, _ netproxy.Dialer, link string) (netproxy.Dialer, *outbounddialer.Property, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return nil, nil, err
	}
	return stubDialer{}, &outbounddialer.Property{
		Name:     parsed.Fragment,
		Protocol: parsed.Scheme,
		Link:     link,
	}, nil
}

func TestNewFromClash(t *testing.T) {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte("name: test\nserver: 192.0.2.1\nport: 443\npassword: secret\nsni: proxy.example\nskip-cert-verify: true\nudp: false\n"), &node); err != nil {
		t.Fatal(err)
	}
	got, err := NewFromClash(&node, &dialer.GlobalOption{}, "test", constructor)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name() != "test" || got.Protocol() != "test" || got.SupportUDP() {
		t.Fatalf("metadata = (%q, %q, udp=%v)", got.Name(), got.Protocol(), got.SupportUDP())
	}
	parsed, err := url.Parse(got.Link())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.User == nil || parsed.User.Username() != "secret" || parsed.Query().Get("sni") != "proxy.example" || parsed.Query().Get("insecure") != "1" {
		t.Fatalf("link = %q", got.Link())
	}
}

func TestNewFromClashWithoutPassword(t *testing.T) {
	var node yaml.Node
	if err := yaml.Unmarshal([]byte("name: test\nserver: 192.0.2.1\nport: 443\n"), &node); err != nil {
		t.Fatal(err)
	}
	got, err := NewFromClash(&node, nil, "test", constructor)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(got.Link())
	if err != nil {
		t.Fatal(err)
	}
	if parsed.User != nil {
		t.Fatalf("unexpected empty user info in link %q", got.Link())
	}
}
