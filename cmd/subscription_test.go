package cmd

import (
	"encoding/base64"
	"testing"

	"github.com/mzz2017/gg/dialer"
	_ "github.com/mzz2017/gg/dialer/hysteria2"
)

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
