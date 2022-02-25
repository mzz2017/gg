package grpc

import (
	"fmt"
	"golang.org/x/net/proxy"
	"net"
	"net/url"
)

// Grpc is a base Grpc struct
type Grpc struct {
	dialer    proxy.Dialer
	gunConfig Config
}

// NewGrpc returns a Grpc infra.
func NewGrpc(s string, d proxy.Dialer) (*Grpc, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("NewGrpc: %w", err)
	}

	t := &Grpc{
		dialer: d,
	}

	query := u.Query()
	t.gunConfig.ServerName = query.Get("sni")
	t.gunConfig.ServiceName = query.Get("serviceName")

	if t.gunConfig.ServerName == "" {
		t.gunConfig.ServerName = u.Hostname()
	}

	return t, nil
}

func (s *Grpc) Dial(network, addr string) (conn net.Conn, err error) {
	conf := s.gunConfig
	conf.RemoteAddr = addr
	if conn, err = NewGunClient(&conf).DialConn(); err != nil {
		return nil, err
	}
	return conn, err
}
