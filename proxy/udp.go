package proxy

import (
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/pool"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/protocol/shadowsocks"
	"github.com/mzz2017/gg/infra/ip_mtu_trie"
	"golang.org/x/net/dns/dnsmessage"
	"inet.af/netaddr"
	"net"
	"strings"
	"time"
)

const (
	DefaultNatTimeout = 3 * time.Minute
	DnsQueryTimeout   = 17 * time.Second // RFC 5452
)

type HijackResp struct {
	Resp   []byte
	Domain string
	Type   dnsmessage.Type
	AnsIP  netaddr.IP
}

func (p *Proxy) handleUDP(lAddr net.Addr, data []byte) (err error) {
	loopback, _ := netaddr.FromStdIP(lAddr.(*net.UDPAddr).IP)
	tgt := p.GetProjection(loopback)
	if tgt == "" {
		return fmt.Errorf("mapped target address not found")
	}
	p.log.Tracef("received udp: %v, tgt: %v", lAddr.String(), tgt)
	if hijackResp, isDNSQuery := p.hijackDNS(data); isDNSQuery {
		if hijackResp != nil {
			switch hijackResp.Type {
			case dnsmessage.TypeAAAA:
				// TODO: support to restore INET6 ICMP target
				_, err = p.udpConn.WriteTo(hijackResp.Resp, lAddr)
				return err
			case dnsmessage.TypeA:
				respData, respMsg, err := forwardDNSMessage(tgt, data)
				if err != nil {
					return fmt.Errorf("forwardDNSMessage: %w", err)
				}
				if len(respMsg.Answers) == 0 {
					// no answer
					_, err = p.udpConn.WriteTo(respData, lAddr)
					return err
				}
				// we only pick the first answer
				realAnsA, okA := respMsg.Answers[0].Body.(*dnsmessage.AResource)
				if !okA {
					// not a valid answer
					_, err = p.udpConn.WriteTo(respData, lAddr)
					return err
				}
				ip := netaddr.IPFrom4(realAnsA.A)
				p.realIPMapper.Set(hijackResp.AnsIP, ip)
				p.log.Tracef("fakeIP:(%v) realIP:(%v)", hijackResp.AnsIP, ip)
				_, err = p.udpConn.WriteTo(hijackResp.Resp, lAddr)
				return err
			}
			// TODO: try to send from original address if the socket uses bind.
			// 		But to archive it, we need bind permission.
			//		Is it worth it?
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
	//p.log.Tracef("writeto: %v, %v", targetAddr, data)
	if _, err = rc.WriteTo(data, targetAddr); err != nil {
		return fmt.Errorf("write error: %w", err)
	}
	return nil
}

func (p *Proxy) hijackDNS(data []byte) (resp *HijackResp, isDNSQuery bool) {
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
	var domain string
	var ans netaddr.IP
	switch q.Type {
	case dnsmessage.TypeAAAA:
		domain = strings.TrimSuffix(q.Name.String(), ".")
		ans = p.AllocProjection(domain)
	case dnsmessage.TypeA:
		domain = strings.TrimSuffix(q.Name.String(), ".")
		ans = p.AllocProjection(domain)
		dmsg.Answers = []dnsmessage.Resource{{
			Header: dnsmessage.ResourceHeader{
				Name:  q.Name,
				Class: q.Class,
				TTL:   10,
			},
			Body: &dnsmessage.AResource{A: ans.As4()},
		}}
	}
	switch q.Type {
	case dnsmessage.TypeA, dnsmessage.TypeAAAA:
		p.log.Tracef("hijackDNS: lookup: %v to %v", q.Name.String(), ans.String())
		dmsg.RCode = dnsmessage.RCodeSuccess
		dmsg.Response = true
		dmsg.RecursionAvailable = true
		dmsg.Truncated = false
		b, _ := dmsg.Pack()
		return &HijackResp{
			Resp:   b,
			Domain: domain,
			Type:   q.Type,
			AnsIP:  ans,
		}, true
	}
	return nil, true
}

// SelectTimeout selects an appropriate timeout for UDP packet.
func SelectTimeout(packet []byte) time.Duration {
	var dMessage dnsmessage.Message
	if err := dMessage.Unpack(packet); err != nil {
		return DefaultNatTimeout
	}
	return DnsQueryTimeout
}

// select an appropriate timeout
func selectTimeout(packet []byte) time.Duration {
	al, _ := shadowsocks.BytesSizeForMetadata(packet)
	if len(packet) < al {
		// err: packet with inadequate length
		return DefaultNatTimeout
	}
	packet = packet[al:]
	return SelectTimeout(packet)
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
			if e := p.relayUDP(lAddr, rc, conn.Timeout); e != nil {
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

func (p *Proxy) relayUDP(laddr net.Addr, rConn net.PacketConn, timeout time.Duration) (err error) {
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
		//var dmsg dnsmessage.Message
		//if err := dmsg.Unpack(buf[:n]); err == nil {
		//	p.log.Traceln(dmsg)
		//}
		_ = p.udpConn.SetWriteDeadline(time.Now().Add(DefaultNatTimeout)) // should keep consistent
		_, err = p.udpConn.WriteTo(buf[:n], laddr)
		if err != nil {
			return
		}
	}
}

func forwardDNSMessage(tgt string, msg []byte) ([]byte, *dnsmessage.Message, error) {
	conn, err := net.Dial("udp", tgt)
	if err != nil {
		return nil, nil, err
	}
	_, err = conn.Write(msg)
	if err != nil {
		return nil, nil, err
	}
	buf := make([]byte, 2+512) // see RFC 1035
	n, err := conn.Read(buf)
	if err != nil {
		return nil, nil, err
	}
	var resp dnsmessage.Message
	if err = resp.Unpack(buf[:n]); err != nil {
		return nil, nil, err
	}
	return buf, &resp, nil

}
