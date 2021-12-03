package proxy

import (
	"net"
	"sync"
	"time"
)

type UDPConn struct {
	Establishing chan struct{}
	Timeout      time.Duration
	net.PacketConn
}

func NewUDPConn(conn net.PacketConn) *UDPConn {
	c := &UDPConn{
		PacketConn:   conn,
		Establishing: make(chan struct{}),
	}
	if c.PacketConn != nil {
		close(c.Establishing)
	}
	return c
}

type UDPConnMapping struct {
	nm map[string]*UDPConn
	sync.Mutex
}

func NewUDPConnMapping() *UDPConnMapping {
	m := &UDPConnMapping{
		nm: make(map[string]*UDPConn),
	}
	return m
}

func (m *UDPConnMapping) Get(key string) (conn *UDPConn, ok bool) {
	v, ok := m.nm[key]
	if ok {
		conn = v
	}
	return
}

// pass val=nil for stating it is establishing
func (m *UDPConnMapping) Insert(key string, val net.PacketConn) *UDPConn {
	c := NewUDPConn(val)
	m.nm[key] = c
	return c
}

func (m *UDPConnMapping) Remove(key string) {
	v, ok := m.nm[key]
	if !ok {
		return
	}
	select {
	case <-v.Establishing:
		_ = v.Close()
	default:
		close(v.Establishing)
	}
	delete(m.nm, key)
}
