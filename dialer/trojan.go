package dialer

import (
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/protocol"
	"github.com/mzz2017/gg/dialer/transport/tls"
	"github.com/mzz2017/gg/dialer/transport/ws"
	"golang.org/x/net/proxy"
	"net"
	"net/url"
	"strconv"
	"strings"
)

func init() {
	FromLinkRegister("trojan", NewTrojan)
	FromLinkRegister("trojan-go", NewTrojan)
}

type Trojan struct {
	Name          string `json:"name"`
	Server        string `json:"server"`
	Port          int    `json:"port"`
	Password      string `json:"password"`
	Sni           string `json:"sni"`
	Type          string `json:"type"`
	Encryption    string `json:"encryption"`
	Host          string `json:"host"`
	Path          string `json:"path"`
	AllowInsecure bool   `json:"allowInsecure"`
	Protocol      string `json:"protocol"`
}

func NewTrojan(link string) (*Dialer, error) {
	s, err := ParseTrojanURL(link)
	if err != nil {
		return nil, err
	}
	var dialer proxy.Dialer = proxy.Direct
	u := url.URL{
		Scheme: "tls",
		Host:   net.JoinHostPort(s.Server, strconv.Itoa(s.Port)),
		RawQuery: url.Values{
			"sni": []string{s.Sni},
		}.Encode(),
	}
	if dialer, err = tls.NewTls(u.String(), dialer); err != nil {
		return nil, err
	}
	if s.Protocol == "trojan-go" {
		// "tls,ws,ss,trojanc"
		if s.Type == "ws" {
			u = url.URL{
				Scheme: "ws",
				Host:   net.JoinHostPort(s.Server, strconv.Itoa(s.Port)),
				RawQuery: url.Values{
					"host": []string{s.Host},
					"path": []string{s.Path},
				}.Encode(),
			}
			if dialer, err = ws.NewWs(u.String(), dialer); err != nil {
				return nil, err
			}
		}
		if strings.HasPrefix(s.Encryption, "ss;") {
			fields := strings.SplitN(s.Encryption, ";", 3)
			if dialer, err = protocol.NewDialer("shadowsocks", dialer, protocol.Header{
				ProxyAddress: net.JoinHostPort(s.Server, strconv.Itoa(s.Port)),
				Cipher:       fields[1],
				Password:     fields[2],
				IsClient:     false,
			}); err != nil {
				return nil, err
			}
		}
	}
	if dialer, err = protocol.NewDialer("trojanc", dialer, protocol.Header{
		ProxyAddress: net.JoinHostPort(s.Server, strconv.Itoa(s.Port)),
		Password:     s.Password,
		IsClient:     true,
	}); err != nil {
		return nil, err
	}
	return &Dialer{
		Dialer:     dialer,
		supportUDP: true,
		name:       s.Name,
		link:       link,
	}, nil
}

func ParseTrojanURL(u string) (data *Trojan, err error) {
	//trojan://password@server:port#escape(remarks)
	t, err := url.Parse(u)
	if err != nil {
		err = fmt.Errorf("invalid trojan format")
		return
	}
	allowInsecure := t.Query().Get("allowInsecure")
	sni := t.Query().Get("peer")
	if sni == "" {
		sni = t.Query().Get("sni")
	}
	if sni == "" {
		sni = t.Hostname()
	}
	port, err := strconv.Atoi(t.Port())
	if err != nil {
		return nil, InvalidParameterErr
	}
	data = &Trojan{
		Name:          t.Fragment,
		Server:        t.Hostname(),
		Port:          port,
		Password:      t.User.Username(),
		Sni:           sni,
		AllowInsecure: allowInsecure == "1" || allowInsecure == "true",
		Protocol:      "trojan",
	}
	if t.Scheme == "trojan-go" {
		data.Protocol = "trojan-go"
		data.Encryption = t.Query().Get("encryption")
		data.Host = t.Query().Get("host")
		data.Path = t.Query().Get("path")
		data.Type = t.Query().Get("type")
		data.AllowInsecure = false
	}
	return data, nil
}
