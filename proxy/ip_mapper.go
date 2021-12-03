package proxy

import "inet.af/netaddr"

type IPMapper interface {
	Alloc(target string) (loopback netaddr.IP)
	Get(loopback netaddr.IP) (target string)
}

// LoopbackMapper projects something to a loopback IP.
// It is not thread-safe.
type LoopbackMapper struct {
	mapper    map[netaddr.IP]string
	revMapper map[string]netaddr.IP
	lastAlloc netaddr.IP
}

func NewLoopbackMapper() *LoopbackMapper {
	return &LoopbackMapper{
		mapper:    make(map[netaddr.IP]string),
		revMapper: make(map[string]netaddr.IP),
		lastAlloc: netaddr.IPv4(126, 0, 0, 0),
	}
}

func (m *LoopbackMapper) Alloc(target string) (loopback netaddr.IP) {
	if ip, ok := m.revMapper[target]; ok {
		return ip
	}

	m.lastAlloc = m.lastAlloc.Next()
	if !m.lastAlloc.IsLoopback() {
		// The loopback address has been used up. Loop back and overwrite.
		m.lastAlloc = netaddr.IPv4(127, 0, 0, 1)
	}
	m.mapper[m.lastAlloc] = target
	m.revMapper[target] = m.lastAlloc
	return m.lastAlloc
}

func (m *LoopbackMapper) Get(loopback netaddr.IP) (target string) {
	return m.mapper[loopback]
}

var ReservedPrefix = netaddr.MustParseIPPrefix("198.18.0.0/15")

// ReservedMapper projects something to a reserved IP.
// It is not thread-safe.
type ReservedMapper struct {
	mapper    map[netaddr.IP]string
	revMapper map[string]netaddr.IP
	lastAlloc netaddr.IP
}

func NewReservedMapper() *ReservedMapper {
	return &ReservedMapper{
		mapper:    make(map[netaddr.IP]string),
		revMapper: make(map[string]netaddr.IP),
		lastAlloc: netaddr.IPv4(198, 18, 0, 0),
	}
}

func (m *ReservedMapper) Alloc(target string) (loopback netaddr.IP) {
	if ip, ok := m.revMapper[target]; ok {
		return ip
	}

	m.lastAlloc = m.lastAlloc.Next()
	if !ReservedPrefix.Contains(m.lastAlloc) {
		// The loopback address has been used up. Loop back and overwrite.
		m.lastAlloc = netaddr.IPv4(198, 18, 0, 1)
	}
	m.mapper[m.lastAlloc] = target
	m.revMapper[target] = m.lastAlloc
	return m.lastAlloc
}

func (m *ReservedMapper) Get(loopback netaddr.IP) (target string) {
	return m.mapper[loopback]
}
