package ip_mtu_trie

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/infra/trie"
	"net"
	"strconv"
	"strings"
)

const (
	MTU = 65535
)

var MTUTrie *IPMTUTrie

func init() {
	var err error
	MTUTrie, err = NewIPMTUTrieFromInterfaces()
	if err != nil {
		MTUTrie = new(IPMTUTrie)
	}
}

type IPMTUTrie struct {
	v4Trie       *trie.Trie
	v4Prefix2MTU map[string]int
	v6Trie       *trie.Trie
	v6Prefix2MTU map[string]int
}

func (t *IPMTUTrie) GetMTU(ip net.IP) int {
	mtu := MTU
	if ip := ip.To4(); ip != nil {
		if t.v4Trie == nil {
			return mtu
		}
		prefix := t.v4Trie.Match(IPToBin(ip))
		if m, ok := t.v4Prefix2MTU[prefix]; ok && m < mtu {
			mtu = m
		}
	} else {
		if t.v6Trie == nil {
			return mtu
		}
		prefix := t.v6Trie.Match(IPToBin(ip))
		if m, ok := t.v6Prefix2MTU[prefix]; ok && m < mtu {
			mtu = m
		}
	}
	return mtu
}

func NewIPMTUTrieFromInterfaces() (*IPMTUTrie, error) {
	ifces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	var (
		v4dict []string
		v4m    = make(map[string]int)
		v6dict []string
		v6m    = make(map[string]int)
	)
	for _, ifce := range ifces {
		addrs, _ := ifce.Addrs()
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				ones, bits := ipnet.Mask.Size()
				prefix := IPToBin(ipnet.IP)[:ones]
				switch bits {
				case 32:
					v4dict = append(v4dict, prefix)
					v4m[prefix] = ifce.MTU
				case 128:
					v6dict = append(v6dict, prefix)
					v6m[prefix] = ifce.MTU
				}
			}
		}
	}
	return &IPMTUTrie{
		v4Trie:       trie.New(v4dict),
		v4Prefix2MTU: v4m,
		v6Trie:       trie.New(v6dict),
		v6Prefix2MTU: v6m,
	}, nil
}

func IPToBin(ip net.IP) string {
	var buf strings.Builder
	if ip.To4() != nil {
		ip = ip.To4()
	} else {
		ip = ip.To16()
	}
	for _, b := range ip {
		tmp := strconv.FormatInt(int64(b), 2)
		buf.WriteString(strings.Repeat("0", 8-len(tmp)) + tmp)
	}
	return buf.String()
}
