package cmd

import (
	"encoding/base64"
	"testing"

	"github.com/mzz2017/gg/dialer"
	_ "github.com/mzz2017/gg/dialer/anytls"
	_ "github.com/mzz2017/gg/dialer/hysteria2"
)

func TestResolveSubscriptionAsClashAnyTLS(t *testing.T) {
	config := []byte(`proxies:
  - name: clash-anytls
    type: anytls
    server: 192.0.2.1
    port: 443
    password: secret
    sni: proxy.example
    skip-cert-verify: true
    udp: false
`)
	dialers, err := resolveSubscriptionAsClash(NewLogger(0), &dialer.GlobalOption{}, config)
	if err != nil {
		t.Fatal(err)
	}
	if len(dialers) != 1 {
		t.Fatalf("dialers = %d, want 1", len(dialers))
	}
	if dialers[0].Protocol() != "anytls" {
		t.Fatalf("protocol = %q, want anytls", dialers[0].Protocol())
	}
	if dialers[0].SupportUDP() {
		t.Fatal("expected UDP support to be disabled")
	}
}

func TestResolveSubscriptionAsBase64AnyTLS(t *testing.T) {
	link := "anytls://test-auth@192.0.2.1:443?sni=proxy.example&insecure=1#anytls-node"
	subscription := []byte(base64.StdEncoding.EncodeToString([]byte(link)))

	dialers := resolveSubscriptionAsBase64(NewLogger(0), &dialer.GlobalOption{}, subscription)
	if len(dialers) != 1 {
		t.Fatalf("dialers = %d, want 1", len(dialers))
	}
	if got := dialers[0].Protocol(); got != "anytls" {
		t.Fatalf("protocol = %q, want anytls", got)
	}
	if !dialers[0].SupportUDP() {
		t.Fatal("expected UDP support")
	}
}

func TestResolveSubscriptionAsBase64Hysteria2(t *testing.T) {
	link := "hysteria2://secret@192.0.2.1:443?sni=proxy.example&insecure=1#hy2-node"
	subscription := []byte(base64.StdEncoding.EncodeToString([]byte(link)))

	dialers := resolveSubscriptionAsBase64(NewLogger(0), &dialer.GlobalOption{}, subscription)
	if len(dialers) != 1 {
		t.Fatalf("dialers = %d, want 1", len(dialers))
	}
	if got := dialers[0].Protocol(); got != "hysteria2" {
		t.Fatalf("protocol = %q, want hysteria2", got)
	}
	if !dialers[0].SupportUDP() {
		t.Fatal("expected UDP support")
	}
}
