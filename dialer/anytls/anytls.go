// Package anytls adapts outbound's AnyTLS share-link dialer to gg.
package anytls

import (
	outboundanytls "github.com/daeuniverse/outbound/dialer/anytls"
	_ "github.com/daeuniverse/outbound/protocol/anytls"
	"github.com/mzz2017/gg/dialer"
	"github.com/mzz2017/gg/dialer/outboundlink"
	"gopkg.in/yaml.v3"
)

func init() {
	dialer.FromLinkRegister("anytls", New)
	dialer.FromClashRegister("anytls", NewFromClash)
}

// NewFromClash creates an AnyTLS dialer from a Clash proxy entry.
func NewFromClash(node *yaml.Node, option *dialer.GlobalOption) (*dialer.Dialer, error) {
	return outboundlink.NewFromClash(node, option, "anytls", outboundanytls.NewAnytls)
}

// New creates an AnyTLS dialer from a share link.
func New(link string, option *dialer.GlobalOption) (*dialer.Dialer, error) {
	return outboundlink.New(link, option, outboundanytls.NewAnytls)
}
