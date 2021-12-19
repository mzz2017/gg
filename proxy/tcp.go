package proxy

import (
	"errors"
	"fmt"
	io2 "github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/pkg/zeroalloc/io"
	"inet.af/netaddr"
	"net"
	"time"
)

func (p *Proxy) handleTCP(conn net.Conn) error {
	defer conn.Close()
	loopback, _ := netaddr.FromStdIP(conn.LocalAddr().(*net.TCPAddr).IP)
	tgt := p.GetProjection(loopback)
	if tgt == "" {
		return fmt.Errorf("mapped target address not found")
	}
	p.log.Tracef("received tcp: %v, tgt: %v", conn.RemoteAddr().String(), tgt)
	c, err := p.dialer.Dial("tcp", tgt)
	if err != nil {
		return err
	}
	defer c.Close()
	if err = RelayTCP(conn, c); err != nil {
		var netErr net.Error
		if errors.As(err, &netErr) && netErr.Timeout() {
			return nil // ignore i/o timeout
		}
		return fmt.Errorf("handleTCP relay error: %w", err)
	}
	return nil
}


type WriteCloser interface {
	CloseWrite() error
}

func RelayTCP(lConn, rConn net.Conn) (err error) {
	eCh := make(chan error, 1)
	go func() {
		_, e := io2.Copy(rConn, lConn)
		if rConn, ok := rConn.(WriteCloser); ok {
			rConn.CloseWrite()
		}
		rConn.SetReadDeadline(time.Now().Add(10 * time.Second))
		eCh <- e
	}()
	_, e := io2.Copy(lConn, rConn)
	if lConn, ok := lConn.(WriteCloser); ok {
		lConn.CloseWrite()
	}
	lConn.SetReadDeadline(time.Now().Add(10 * time.Second))
	if e != nil {
		<-eCh
		return e
	}
	return <-eCh
}