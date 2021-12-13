package proxy

import (
	"errors"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/pool"
	"github.com/mzz2017/gg/infra/ip_mtu_trie"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
	"inet.af/netaddr"
	"net"
	"strconv"
	"strings"
	"sync"
)

type Proxy struct {
	mutex        sync.Mutex      // mutex protects the mappers
	addrMapper   *LoopbackMapper // addrMapper projects an address to a loopback IP
	domainMapper *ReservedMapper // domainMapper projects a domain to a reserved IP
	realIPMapper *RealIPMapper   // realIPMapper projects a fake IP to a real IP

	log         *logrus.Logger
	listener    net.Listener
	udpConn     *net.UDPConn
	dialer      proxy.Dialer
	closed      chan struct{}
	tcpListened chan struct{}

	nm *UDPConnMapping
}

func New(logger *logrus.Logger, dialer proxy.Dialer) *Proxy {
	return &Proxy{
		addrMapper:   NewLoopbackMapper(),
		domainMapper: NewReservedMapper(),
		realIPMapper: NewRealIPMapper(),
		log:          logger,
		dialer:       dialer,
		closed:       make(chan struct{}),
		tcpListened:  make(chan struct{}),
		nm:           NewUDPConnMapping(),
	}
}

func (p *Proxy) AllocProjection(target string) (loopback netaddr.IP) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if strings.Contains(target, ":") {
		// address
		return p.addrMapper.Alloc(target)
	} else {
		// domain
		return p.domainMapper.Alloc(target)
	}
}

func (p *Proxy) GetProjection(ip netaddr.IP) (target string) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	if ip.IsLoopback() {
		// loopback IP -> target address
		return p.addrMapper.Get(ip)
	} else {
		// reserved IP -> domain
		return p.domainMapper.Get(ip)
	}
}

func (p *Proxy) GetRealIP(fakeIP netaddr.IP) (realIP netaddr.IP, ok bool) {
	return p.realIPMapper.Get(fakeIP)
}

// ListenAndServe will block the goroutine.
func (p *Proxy) ListenAndServe(port int) error {
	addr := net.JoinHostPort("0.0.0.0", strconv.Itoa(port))
	eCh := make(chan error, 2)
	go func() {
		e := p.ListenTCP(addr)
		eCh <- e
	}()
	select {
	case e := <-eCh:
		return e
	case <-p.tcpListened:
		// clear
		addr = net.JoinHostPort("0.0.0.0", strconv.Itoa(p.listener.Addr().(*net.TCPAddr).Port))
		go func() {
			e := p.ListenUDP(addr)
			eCh <- e
		}()
	}
	defer p.Close()
	return <-eCh
}

func (p *Proxy) ListenTCP(addr string) (err error) {
	lt, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}
	p.listener = lt
	close(p.tcpListened)
	for {
		conn, err := lt.Accept()
		if err != nil {
			p.log.Infof("%v", err)
		}
		go func() {
			err := p.handleTCP(conn)
			if err != nil {
				p.log.Infof("handleTCP: %v", err)
			}
		}()
	}
}

func (p *Proxy) ListenUDP(addr string) (err error) {
	_, strPort, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	port, err := strconv.Atoi(strPort)
	if err != nil {
		return err
	}

	lu, err := net.ListenUDP("udp", &net.UDPAddr{Port: port})
	if err != nil {
		return err
	}
	p.udpConn = lu
	var buf [ip_mtu_trie.MTU]byte
	for {
		n, lAddr, err := lu.ReadFrom(buf[:])
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				return nil
			}
			p.log.Infof("ReadFrom: %v", err)
			continue
		}
		data := pool.Get(n)
		copy(data, buf[:n])
		go func() {
			err := p.handleUDP(lAddr, data)
			if err != nil {
				p.log.Infof("handleUDP: %v", err)
			}
			pool.Put(data)
		}()
	}
}

func (p *Proxy) TCPPort() int {
	return p.listener.Addr().(*net.TCPAddr).Port
}

func (p *Proxy) UDPPort() int {
	return p.udpConn.LocalAddr().(*net.UDPAddr).Port
}

func (p *Proxy) Close() error {
	close(p.closed)
	var err error
	if p.listener != nil {
		err = p.listener.Close()
	}
	if p.udpConn != nil {
		err2 := p.udpConn.Close()
		if err == nil {
			err = err2
		}
	}
	return err
}
