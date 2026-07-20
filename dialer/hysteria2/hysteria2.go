// Package hysteria2 adapts outbound's Hysteria2 dialer to gg.
package hysteria2

import (
	outboundhysteria2 "github.com/daeuniverse/outbound/dialer/hysteria2"
	_ "github.com/daeuniverse/outbound/protocol/hysteria2"
	"github.com/mzz2017/gg/dialer"
	"github.com/mzz2017/gg/dialer/outboundlink"
	"gopkg.in/yaml.v3"
)

func init() {
	dialer.FromLinkRegister("hysteria2", New)
	dialer.FromLinkRegister("hy2", New)
	dialer.FromClashRegister("hysteria2", NewFromClash)
}

// New creates a Hysteria2 dialer from a share link.
func New(link string, option *dialer.GlobalOption) (*dialer.Dialer, error) {
	return outboundlink.New(link, option, outboundhysteria2.NewHysteria2)
}

// NewFromClash creates a Hysteria2 dialer from a Clash proxy entry.
func NewFromClash(node *yaml.Node, option *dialer.GlobalOption) (*dialer.Dialer, error) {
	return outboundlink.NewFromClash(node, option, "hysteria2", outboundhysteria2.NewHysteria2)
}
