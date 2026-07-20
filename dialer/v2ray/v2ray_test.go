package v2ray

import (
	"testing"

	_ "github.com/daeuniverse/outbound/protocol/vless"
	_ "github.com/daeuniverse/outbound/protocol/vmess"
)

func TestDialerConstructsOutboundProtocols(t *testing.T) {
	tests := []struct {
		name     string
		protocol string
		flow     string
		network  string
	}{
		{name: "VMess", protocol: "vmess", network: "tcp"},
		{name: "VMess gRPC", protocol: "vmess", network: "grpc"},
		{name: "VLESS", protocol: "vless", network: "tcp"},
		{name: "VLESS Vision", protocol: "vless", flow: "xtls-rprx-vision", network: "tcp"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			v := &V2Ray{
				Add:      "192.0.2.1",
				Port:     "443",
				ID:       "00000000-0000-0000-0000-000000000001",
				Net:      test.network,
				Type:     "none",
				SNI:      "example.com",
				Flow:     test.flow,
				Protocol: test.protocol,
			}
			if _, err := v.Dialer(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
