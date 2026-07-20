// Package outboundlink adapts outbound share-link dialers to gg.
package outboundlink

import (
	"net"
	"net/url"
	"strconv"

	outbounddialer "github.com/daeuniverse/outbound/dialer"
	"github.com/daeuniverse/outbound/netproxy"
	"github.com/mzz2017/gg/dialer"
	"gopkg.in/yaml.v3"
)

// Constructor creates an outbound dialer from a share link.
type Constructor func(*outbounddialer.ExtraOption, netproxy.Dialer, string) (netproxy.Dialer, *outbounddialer.Property, error)

// New creates a gg dialer with an outbound share-link constructor.
func New(link string, option *dialer.GlobalOption, constructor Constructor) (*dialer.Dialer, error) {
	extra := &outbounddialer.ExtraOption{}
	if option != nil {
		extra.AllowInsecure = option.AllowInsecure
	}
	proxyDialer, property, err := constructor(extra, dialer.ToNetproxyDialer(dialer.SymmetricDirect), link)
	if err != nil {
		return nil, err
	}
	exportedLink := property.Link
	if parsed, parseErr := url.Parse(exportedLink); parseErr == nil && parsed.User != nil && parsed.User.Username() == "" {
		parsed.User = nil
		exportedLink = parsed.String()
	}
	return dialer.NewDialer(
		dialer.FromNetproxyDialer(proxyDialer),
		true,
		property.Name,
		property.Protocol,
		exportedLink,
	), nil
}

// NewFromClash creates a gg dialer from the common Clash proxy fields.
func NewFromClash(node *yaml.Node, option *dialer.GlobalOption, scheme string, constructor Constructor) (*dialer.Dialer, error) {
	var proxy struct {
		Name           string `yaml:"name"`
		Server         string `yaml:"server"`
		Port           int    `yaml:"port"`
		Password       string `yaml:"password"`
		SNI            string `yaml:"sni"`
		SkipCertVerify bool   `yaml:"skip-cert-verify"`
		UDP            *bool  `yaml:"udp,omitempty"`
	}
	if err := node.Decode(&proxy); err != nil {
		return nil, err
	}
	query := url.Values{}
	if proxy.SNI != "" {
		query.Set("sni", proxy.SNI)
	}
	if proxy.SkipCertVerify {
		query.Set("insecure", "1")
	}
	var user *url.Userinfo
	if proxy.Password != "" {
		user = url.User(proxy.Password)
	}
	link := url.URL{
		Scheme:   scheme,
		Host:     net.JoinHostPort(proxy.Server, strconv.Itoa(proxy.Port)),
		User:     user,
		RawQuery: query.Encode(),
		Fragment: proxy.Name,
	}
	result, err := New(link.String(), option, constructor)
	if err != nil {
		return nil, err
	}
	supportUDP := true
	if proxy.UDP != nil {
		supportUDP = *proxy.UDP
	}
	if supportUDP == result.SupportUDP() {
		return result, nil
	}
	return dialer.NewDialer(result.Dialer, supportUDP, result.Name(), result.Protocol(), result.Link()), nil
}
