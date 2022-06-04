package proxy

import (
	"net/netip"
	"sync"
)

type RealIPMapper struct {
	mapper map[netip.Addr]netip.Addr
	mutex  sync.Mutex
}

func NewRealIPMapper() *RealIPMapper {
	return &RealIPMapper{
		mapper: make(map[netip.Addr]netip.Addr),
	}
}

func (m *RealIPMapper) Set(fakeIP netip.Addr, realIP netip.Addr) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.mapper[fakeIP] = realIP
}

func (m *RealIPMapper) Get(fakeIP netip.Addr) (realIP netip.Addr, ok bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	realIP, ok = m.mapper[fakeIP]
	return
}
