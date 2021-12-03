package proxy

import (
	"errors"
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/server"
	"inet.af/netaddr"
	"net"
)

func (p *Proxy) handleTCP(conn net.Conn) error {
	defer conn.Close()
	loopback, _ := netaddr.FromStdIP(conn.LocalAddr().(*net.TCPAddr).IP)
	tgt := p.GetProjection(loopback)
	if tgt == "" {
		return fmt.Errorf("mapped target address not found")
	}
	c, err := p.dialer.Dial("tcp", tgt)
	if err != nil {
		return err
	}
	defer c.Close()
	if err = server.RelayTCP(conn, c); err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return nil // ignore i/o timeout
		}
		return fmt.Errorf("handleConn relay error: %w", err)
	}
	return nil
}
