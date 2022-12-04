package tls

import (
	"crypto/tls"
	"fmt"
	"github.com/mzz2017/gg/common"
	"golang.org/x/net/proxy"
	"net"
	"net/url"
)

// Tls is a base Tls struct
type Tls struct {
	dialer     proxy.Dialer
	addr       string
	serverName string
	skipVerify bool
	tlsConfig  *tls.Config
}

// NewTls returns a Tls infra.
func NewTls(s string, d proxy.Dialer) (*Tls, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("NewTls: %w", err)
	}

	t := &Tls{
		dialer: d,
		addr:   u.Host,
	}

	query := u.Query()
	t.serverName = query.Get("sni")

	// skipVerify
	if common.StringToBool(query.Get("allowInsecure")) ||
		common.StringToBool(query.Get("skipVerify")) {
		t.skipVerify = true
	}
	if t.serverName == "" {
		t.serverName = u.Hostname()
	}
	t.tlsConfig = &tls.Config{
		ServerName:         t.serverName,
		InsecureSkipVerify: t.skipVerify,
	}

	return t, nil
}

func (s *Tls) Dial(network, addr string) (conn net.Conn, err error) {
	rc, err := s.dialer.Dial("tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("[Tls]: dial to %s: %w", s.addr, err)
	}

	tlsConn := tls.Client(rc, s.tlsConfig)
	if err := tlsConn.Handshake(); err != nil {
		return nil, err
	}
	return tlsConn, err
}
