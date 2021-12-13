package proxy

import (
	"inet.af/netaddr"
	"sync"
)

type RealIPMapper struct {
	mapper map[netaddr.IP]netaddr.IP
	mutex  sync.Mutex
}

func NewRealIPMapper() *RealIPMapper {
	return &RealIPMapper{
		mapper: make(map[netaddr.IP]netaddr.IP),
	}
}

func (m *RealIPMapper) Set(fakeIP netaddr.IP, realIP netaddr.IP) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.mapper[fakeIP] = realIP
}

func (m *RealIPMapper) Get(fakeIP netaddr.IP) (realIP netaddr.IP, ok bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	realIP, ok = m.mapper[fakeIP]
	return
}
