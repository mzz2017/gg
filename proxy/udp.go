package proxy

import (
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/pool"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/protocol/shadowsocks"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/server"
	"golang.org/x/net/dns/dnsmessage"
	"inet.af/netaddr"
	"net"
	"ptrace/infra/ip_mtu_trie"
	"time"
)

func (p *Proxy) handleUDP(lAddr net.Addr, data []byte) (err error) {
	loopback, _ := netaddr.FromStdIP(lAddr.(*net.UDPAddr).IP)
	tgt := p.GetProjection(loopback)
	if tgt == "" {
		return fmt.Errorf("mapped target address not found")
	}
	p.log.Tracef("received udp: %v, tgt: %v", lAddr.String(), tgt)
	if resp, isDNSQuery := p.hijackDNS(data); isDNSQuery {
		if resp != nil {
			_, err = p.udpConn.WriteTo(resp, lAddr)
			return err

			// TODO: try to send from original address if the socket uses bind.
			// 		But to archive it, we need bind permission.
			//		Is it worth it?

			//host, strPort, err := net.SplitHostPort(tgt)
			//if err != nil {
			//	return err
			//}
			//ip, err := netaddr.ParseIP(host)
			//if err != nil {
			//	return err
			//}
			//port, err := strconv.Atoi(strPort)
			//if err != nil {
			//	return err
			//}
			//p.log.Warnf("send from: %v", netaddr.IPPortFrom(ip, uint16(port)))
			//conn, err := ptrace.NewUDPDialer(netaddr.IPPortFrom(ip, uint16(port)), 10*time.Second, p.log).Dial("udp", lAddr.String())
			//if err != nil {
			//	return err
			//}
			//_, err = conn.Write(resp)
			//return err
		}
		// continue to request DNS but use replaced DNS server.
		tgt = "1.1.1.1:53"
	}
	rc, err := p.GetOrBuildUDPConn(lAddr, tgt, data)
	if err != nil {
		return fmt.Errorf("auth fail from: %v: %w", lAddr.String(), err)
	}
	targetAddr, err := net.ResolveUDPAddr("udp", tgt)
	if err != nil {
		return err
	}
	p.log.Tracef("writeto: %v, %v", targetAddr, data)
	if _, err = rc.WriteTo(data, targetAddr); err != nil {
		return fmt.Errorf("write error: %w", err)
	}
	return nil
}

func (p *Proxy) hijackDNS(data []byte) (resp []byte, isDNSQuery bool) {
	var dmsg dnsmessage.Message
	if dmsg.Unpack(data) != nil {
		return nil, false
	}
	if len(dmsg.Questions) == 0 {
		return nil, true
	}
	// we only peek the first question.
	// see https://stackoverflow.com/questions/4082081/requesting-a-and-aaaa-records-in-single-dns-query/4083071#4083071
	q := dmsg.Questions[0]
	switch q.Type {
	case dnsmessage.TypeAAAA:
		// empty answer
		dmsg.RCode = dnsmessage.RCodeSuccess
		dmsg.Response = true
		dmsg.RecursionAvailable = true
		dmsg.Truncated = false
		b, _ := dmsg.Pack()
		return b, true
	case dnsmessage.TypeA:
		ans := p.AllocProjection(q.Name.String()).As4()
		dmsg.Answers = []dnsmessage.Resource{{
			Header: dnsmessage.ResourceHeader{
				Name:  q.Name,
				Class: q.Class,
				TTL:   10,
			},
			Body: &dnsmessage.AResource{A: ans},
		}}
		p.log.Tracef("hijackDNS: lookup: %v to %v", q.Name.String(), ans)
		dmsg.RCode = dnsmessage.RCodeSuccess
		dmsg.Response = true
		dmsg.RecursionAvailable = true
		dmsg.Truncated = false
		b, _ := dmsg.Pack()
		return b, true
	}
	return nil, true
}

// select an appropriate timeout
func selectTimeout(packet []byte) time.Duration {
	al, _ := shadowsocks.BytesSizeForMetadata(packet)
	if len(packet) < al {
		// err: packet with inadequate length
		return server.DefaultNatTimeout
	}
	packet = packet[al:]
	return server.SelectTimeout(packet)
}

// GetOrBuildUDPConn get a UDP conn from the mapping.
func (p *Proxy) GetOrBuildUDPConn(lAddr net.Addr, target string, data []byte) (rc net.PacketConn, err error) {
	var conn *UDPConn
	var ok bool

	connIdent := lAddr.String()
	p.nm.Lock()
	if conn, ok = p.nm.Get(connIdent); !ok {
		// not exist such socket mapping, build one
		p.nm.Insert(connIdent, nil)
		p.nm.Unlock()

		// dial
		c, err := p.dialer.Dial("udp", target)
		if err != nil {
			p.nm.Lock()
			p.nm.Remove(connIdent) // close channel to inform that establishment ends
			p.nm.Unlock()
			return nil, fmt.Errorf("GetOrBuildUDPConn dial error: %w", err)
		}
		rc = c.(net.PacketConn)
		p.nm.Lock()
		p.nm.Remove(connIdent) // close channel to inform that establishment ends
		conn = p.nm.Insert(connIdent, rc)
		conn.Timeout = selectTimeout(data)
		p.nm.Unlock()
		// relay
		go func() {
			if e := p.relay(lAddr, rc, conn.Timeout); e != nil {
				p.log.Tracef("shadowsocks.udp.relay: %v", e)
			}
			p.nm.Lock()
			p.nm.Remove(connIdent)
			p.nm.Unlock()
		}()
	} else {
		// such socket mapping exists; just verify or wait for its establishment
		p.nm.Unlock()
		<-conn.Establishing
		if conn.PacketConn == nil {
			// establishment ended and retrieve the result
			return p.GetOrBuildUDPConn(lAddr, target, data)
		} else {
			// establishment succeeded
			rc = conn.PacketConn
		}
	}
	// countdown
	_ = conn.PacketConn.SetReadDeadline(time.Now().Add(conn.Timeout))
	return rc, nil
}

func (p *Proxy) relay(laddr net.Addr, rConn net.PacketConn, timeout time.Duration) (err error) {
	buf := pool.Get(ip_mtu_trie.MTUTrie.GetMTU(rConn.LocalAddr().(*net.UDPAddr).IP))
	defer pool.Put(buf)
	var n int
	for {
		p.log.Tracef("readfrom...")
		_ = rConn.SetReadDeadline(time.Now().Add(timeout))
		n, _, err = rConn.ReadFrom(buf)
		if err != nil {
			return fmt.Errorf("rConn.ReadFrom: %v", err)
		}
		p.log.Tracef("readfrom: %v", buf[:n])
		_ = p.udpConn.SetWriteDeadline(time.Now().Add(server.DefaultNatTimeout)) // should keep consistent
		_, err = p.udpConn.WriteTo(buf[:n], laddr)
		if err != nil {
			return
		}
	}
}
