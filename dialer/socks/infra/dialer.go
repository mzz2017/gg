package infra

import (
	"h12.io/socks"
	"net"
)

type Dialer struct {
	dial func(string, string) (net.Conn, error)
}

func NewDialer(link string) *Dialer {
	return &Dialer{dial: socks.Dial(link)}
}

func (d *Dialer) Dial(network, addr string) (net.Conn, error) {
	return d.dial(network, addr)
}
