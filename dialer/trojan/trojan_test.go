package trojan

import (
	"testing"

	_ "github.com/daeuniverse/outbound/protocol/trojanc"
)

func TestDialerUsesOutbound(t *testing.T) {
	for name, transport := range map[string]string{"tcp": "", "grpc": "grpc"} {
		t.Run(name, func(t *testing.T) {
			trojan := &Trojan{
				Server:      "192.0.2.1",
				Port:        443,
				Password:    "password",
				Sni:         "example.com",
				Type:        transport,
				ServiceName: "GunService",
				Protocol:    "trojan",
			}
			if _, err := trojan.Dialer(); err != nil {
				t.Fatal(err)
			}
		})
	}
}
