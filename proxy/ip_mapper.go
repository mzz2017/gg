package proxy

import (
	"net/netip"
)

type IPMapper interface {
	Alloc(target string) (loopback netip.Addr)
	Get(loopback netip.Addr) (target string)
}

// LoopbackMapper projects something to a loopback IP.
// It is not thread-safe.
type LoopbackMapper struct {
	mapper    map[netip.Addr]string
	revMapper map[string]netip.Addr
	lastAlloc netip.Addr
}

func NewLoopbackMapper() *LoopbackMapper {
	return &LoopbackMapper{
		mapper:    make(map[netip.Addr]string),
		revMapper: make(map[string]netip.Addr),
		lastAlloc: netip.AddrFrom4([4]byte{126, 0, 0, 0}),
	}
}

func (m *LoopbackMapper) Alloc(target string) (loopback netip.Addr) {
	if ip, ok := m.revMapper[target]; ok {
		return ip
	}

	m.lastAlloc = m.lastAlloc.Next()
	if !m.lastAlloc.IsLoopback() {
		// The loopback address has been used up. Loop back and overwrite.
		m.lastAlloc = netip.AddrFrom4([4]byte{127, 0, 0, 1})
	}
	m.mapper[m.lastAlloc] = target
	m.revMapper[target] = m.lastAlloc
	return m.lastAlloc
}

func (m *LoopbackMapper) Get(loopback netip.Addr) (target string) {
	return m.mapper[loopback]
}

var ReservedPrefix = netip.MustParsePrefix("198.18.0.0/15")

// ReservedMapper projects something to a reserved IP.
// It is not thread-safe.
type ReservedMapper struct {
	mapper    map[netip.Addr]string
	revMapper map[string]netip.Addr
	lastAlloc netip.Addr
}

func NewReservedMapper() *ReservedMapper {
	return &ReservedMapper{
		mapper:    make(map[netip.Addr]string),
		revMapper: make(map[string]netip.Addr),
		lastAlloc: netip.AddrFrom4([4]byte{198, 18, 0, 0}),
	}
}

func (m *ReservedMapper) Alloc(target string) (loopback netip.Addr) {
	if ip, ok := m.revMapper[target]; ok {
		return ip
	}

	m.lastAlloc = m.lastAlloc.Next()
	if !ReservedPrefix.Contains(m.lastAlloc) {
		// The loopback address has been used up. Loop back and overwrite.
		m.lastAlloc = netip.AddrFrom4([4]byte{198, 18, 0, 1})
	}
	m.mapper[m.lastAlloc] = target
	m.revMapper[target] = m.lastAlloc
	return m.lastAlloc
}

func (m *ReservedMapper) Get(loopback netip.Addr) (target string) {
	return m.mapper[loopback]
}
