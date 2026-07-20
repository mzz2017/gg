package dialer

import (
	"context"
	"errors"
	"io"
	"net"
	"testing"
	"time"

	"github.com/daeuniverse/outbound/netproxy"
	"golang.org/x/net/proxy"
)

type proxyDialerFunc func(network, addr string) (net.Conn, error)

func (f proxyDialerFunc) Dial(network, addr string) (net.Conn, error) {
	return f(network, addr)
}

type netproxyDialerFunc func(context.Context, string, string) (netproxy.Conn, error)

func (f netproxyDialerFunc) DialContext(ctx context.Context, network, addr string) (netproxy.Conn, error) {
	return f(ctx, network, addr)
}

type connWithoutAddr struct {
	conn net.Conn
}

func (c *connWithoutAddr) Read(p []byte) (int, error)         { return c.conn.Read(p) }
func (c *connWithoutAddr) Write(p []byte) (int, error)        { return c.conn.Write(p) }
func (c *connWithoutAddr) Close() error                       { return c.conn.Close() }
func (c *connWithoutAddr) SetDeadline(t time.Time) error      { return c.conn.SetDeadline(t) }
func (c *connWithoutAddr) SetReadDeadline(t time.Time) error  { return c.conn.SetReadDeadline(t) }
func (c *connWithoutAddr) SetWriteDeadline(t time.Time) error { return c.conn.SetWriteDeadline(t) }

type packetConnStub struct {
	readData  []byte
	readAddr  net.Addr
	writeData []byte
	writeAddr net.Addr
}

func (c *packetConnStub) Read(p []byte) (int, error) {
	n, _, err := c.ReadFrom(p)
	return n, err
}

func (c *packetConnStub) Write(p []byte) (int, error) {
	return c.WriteTo(p, c.RemoteAddr())
}

func (c *packetConnStub) Close() error                     { return nil }
func (c *packetConnStub) LocalAddr() net.Addr              { return &net.UDPAddr{} }
func (c *packetConnStub) RemoteAddr() net.Addr             { return c.readAddr }
func (c *packetConnStub) SetDeadline(time.Time) error      { return nil }
func (c *packetConnStub) SetReadDeadline(time.Time) error  { return nil }
func (c *packetConnStub) SetWriteDeadline(time.Time) error { return nil }
func (c *packetConnStub) ReadFrom(p []byte) (int, net.Addr, error) {
	if len(c.readData) == 0 {
		return 0, nil, io.EOF
	}
	n := copy(p, c.readData)
	c.readData = c.readData[n:]
	return n, c.readAddr, nil
}

func (c *packetConnStub) WriteTo(p []byte, addr net.Addr) (int, error) {
	c.writeData = append(c.writeData, p...)
	c.writeAddr = addr
	return len(p), nil
}

func TestNetproxyAdapterTCP(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
	})

	dialer := ToNetproxyDialer(proxyDialerFunc(func(network, addr string) (net.Conn, error) {
		if network != "tcp" {
			t.Fatalf("network = %q, want tcp", network)
		}
		if addr != "example.com:443" {
			t.Fatalf("addr = %q, want example.com:443", addr)
		}
		return clientConn, nil
	}))

	conn, err := dialer.DialContext(context.Background(), "tcp", "example.com:443")
	if err != nil {
		t.Fatal(err)
	}
	if conn != clientConn {
		t.Fatalf("conn = %T, want original pipe connection", conn)
	}
}

func TestProxyAdapterTCP(t *testing.T) {
	clientConn, serverConn := net.Pipe()
	t.Cleanup(func() {
		_ = clientConn.Close()
		_ = serverConn.Close()
	})

	dialer := FromNetproxyDialer(netproxyDialerFunc(func(ctx context.Context, network, addr string) (netproxy.Conn, error) {
		return &connWithoutAddr{conn: clientConn}, nil
	}))
	conn, err := dialer.Dial("tcp", "192.0.2.1:443")
	if err != nil {
		t.Fatal(err)
	}
	if got := conn.RemoteAddr().String(); got != "192.0.2.1:443" {
		t.Fatalf("remote address = %q, want 192.0.2.1:443", got)
	}
}

func TestNetproxyAdaptersUDP(t *testing.T) {
	serverAddr := &net.UDPAddr{IP: net.IPv4(192, 0, 2, 1), Port: 53}
	underlying := &packetConnStub{
		readData: []byte("pong"),
		readAddr: serverAddr,
	}

	baseDialer := proxyDialerFunc(func(network, addr string) (net.Conn, error) {
		if network != "udp" {
			t.Fatalf("network = %q, want udp", network)
		}
		return underlying, nil
	})
	dialer := FromNetproxyDialer(ToNetproxyDialer(baseDialer))
	conn, err := dialer.Dial("udp", serverAddr.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = conn.Close() })

	packetConn, ok := conn.(net.PacketConn)
	if !ok {
		t.Fatalf("conn = %T, want net.PacketConn", conn)
	}
	if err := conn.SetDeadline(time.Now().Add(2 * time.Second)); err != nil {
		t.Fatal(err)
	}

	request := []byte("ping")
	if _, err := packetConn.WriteTo(request, serverAddr); err != nil {
		t.Fatal(err)
	}
	if got := string(underlying.writeData); got != string(request) {
		t.Fatalf("request = %q, want %q", got, request)
	}
	if underlying.writeAddr.String() != serverAddr.String() {
		t.Fatalf("request address = %q, want %q", underlying.writeAddr, serverAddr)
	}

	buf := make([]byte, 32)
	n, addr, err := packetConn.ReadFrom(buf)
	if err != nil {
		t.Fatal(err)
	}
	if got := string(buf[:n]); got != "pong" {
		t.Fatalf("response = %q, want pong", got)
	}
	if addr.String() != serverAddr.String() {
		t.Fatalf("response address = %q, want %q", addr, serverAddr)
	}
}

func TestNetproxyAdapterHonorsCancellation(t *testing.T) {
	unblock := make(chan struct{})
	baseDialer := proxyDialerFunc(func(network, addr string) (net.Conn, error) {
		<-unblock
		return nil, errors.New("dial stopped")
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := ToNetproxyDialer(baseDialer).DialContext(ctx, "tcp", "example.com:443")
	close(unblock)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("error = %v, want context.Canceled", err)
	}
}

var _ proxy.Dialer = proxyDialerFunc(nil)
var _ netproxy.Dialer = netproxyDialerFunc(nil)
