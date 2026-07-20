package dialer

import (
	"context"
	"fmt"
	"net"
	"net/netip"
	"time"

	"github.com/daeuniverse/outbound/netproxy"
	"golang.org/x/net/proxy"
)

// ToNetproxyDialer adapts the project's dialer chain to outbound's dialer API.
func ToNetproxyDialer(d proxy.Dialer) netproxy.Dialer {
	return &netproxyAdapter{dialer: d}
}

// FromNetproxyDialer adapts outbound's result back to the API used by gg.
func FromNetproxyDialer(d netproxy.Dialer) proxy.Dialer {
	return &proxyAdapter{dialer: d}
}

type netproxyAdapter struct {
	dialer proxy.Dialer
}

func (d *netproxyAdapter) DialContext(ctx context.Context, network, addr string) (netproxy.Conn, error) {
	magicNetwork, err := netproxy.ParseMagicNetwork(network)
	if err != nil {
		return nil, err
	}

	conn, err := dialContext(ctx, d.dialer, magicNetwork.Network, addr)
	if err != nil {
		return nil, err
	}
	if magicNetwork.Network != "udp" {
		return conn, nil
	}

	packetConn, ok := conn.(net.PacketConn)
	if !ok {
		_ = conn.Close()
		return nil, fmt.Errorf("UDP dialer returned %T, which does not implement net.PacketConn", conn)
	}
	return &netproxyPacketConn{conn: conn, packetConn: packetConn}, nil
}

func dialContext(ctx context.Context, d proxy.Dialer, network, addr string) (net.Conn, error) {
	if d, ok := d.(interface {
		DialContext(context.Context, string, string) (net.Conn, error)
	}); ok {
		return d.DialContext(ctx, network, addr)
	}

	type result struct {
		conn net.Conn
		err  error
	}
	resultCh := make(chan result, 1)
	go func() {
		conn, err := d.Dial(network, addr)
		resultCh <- result{conn: conn, err: err}
	}()

	select {
	case <-ctx.Done():
		go func() {
			result := <-resultCh
			if result.conn != nil {
				_ = result.conn.Close()
			}
		}()
		return nil, ctx.Err()
	case result := <-resultCh:
		return result.conn, result.err
	}
}

type netproxyPacketConn struct {
	conn       net.Conn
	packetConn net.PacketConn
}

func (c *netproxyPacketConn) Read(p []byte) (int, error)  { return c.conn.Read(p) }
func (c *netproxyPacketConn) Write(p []byte) (int, error) { return c.conn.Write(p) }
func (c *netproxyPacketConn) Close() error                { return c.conn.Close() }

func (c *netproxyPacketConn) SetDeadline(t time.Time) error {
	return c.conn.SetDeadline(t)
}

func (c *netproxyPacketConn) SetReadDeadline(t time.Time) error {
	return c.conn.SetReadDeadline(t)
}

func (c *netproxyPacketConn) SetWriteDeadline(t time.Time) error {
	return c.conn.SetWriteDeadline(t)
}

func (c *netproxyPacketConn) ReadFrom(p []byte) (int, netip.AddrPort, error) {
	n, addr, err := c.packetConn.ReadFrom(p)
	if err != nil {
		return n, netip.AddrPort{}, err
	}
	if udpAddr, ok := addr.(*net.UDPAddr); ok {
		return n, udpAddr.AddrPort(), nil
	}
	addrPort, err := netip.ParseAddrPort(addr.String())
	return n, addrPort, err
}

func (c *netproxyPacketConn) WriteTo(p []byte, addr string) (int, error) {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return 0, err
	}
	return c.packetConn.WriteTo(p, udpAddr)
}

type proxyAdapter struct {
	dialer netproxy.Dialer
}

func (d *proxyAdapter) Dial(network, addr string) (net.Conn, error) {
	return d.DialContext(context.Background(), network, addr)
}

func (d *proxyAdapter) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	conn, err := d.dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	switch network {
	case "tcp":
		if conn, ok := conn.(net.Conn); ok {
			return conn, nil
		}
		remoteAddr, err := net.ResolveTCPAddr("tcp", addr)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		return &netproxy.FakeNetConn{
			Conn:  conn,
			LAddr: &net.TCPAddr{},
			RAddr: remoteAddr,
		}, nil
	case "udp":
		packetConn, ok := conn.(netproxy.PacketConn)
		if !ok {
			_ = conn.Close()
			return nil, fmt.Errorf("outbound UDP dialer returned %T, which does not implement netproxy.PacketConn", conn)
		}
		remoteAddr, err := net.ResolveUDPAddr("udp", addr)
		if err != nil {
			_ = conn.Close()
			return nil, err
		}
		return netproxy.NewFakeNetPacketConn(packetConn, &net.UDPAddr{}, remoteAddr), nil
	default:
		_ = conn.Close()
		return nil, net.UnknownNetworkError(network)
	}
}
