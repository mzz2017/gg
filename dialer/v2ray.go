package dialer

import (
	"fmt"
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/protocol"
	jsoniter "github.com/json-iterator/go"
	"github.com/mzz2017/gg/common"
	"golang.org/x/net/proxy"
	"net"
	"net/url"
	"regexp"
	"strings"
)

func init() {
	FromLinkRegister("vmess", NewV2Ray)
	FromLinkRegister("vless", NewV2Ray)
}

type V2Ray struct {
	Ps            string `json:"ps"`
	Add           string `json:"add"`
	Port          string `json:"port"`
	ID            string `json:"id"`
	Aid           string `json:"aid"`
	Net           string `json:"net"`
	Type          string `json:"type"`
	Host          string `json:"host"`
	SNI           string `json:"sni"`
	Path          string `json:"path"`
	TLS           string `json:"tls"`
	Flow          string `json:"flow,omitempty"`
	Alpn          string `json:"alpn,omitempty"`
	AllowInsecure bool   `json:"allowInsecure"`
	V             string `json:"v"`
	Protocol      string `json:"protocol"`
}

func NewV2Ray(link string) (*Dialer, error) {
	if strings.HasPrefix(link, "vmess://") {
		s, err := ParseVmessURL(link)
		if err != nil {
			return nil, err
		}
		if s.Net != "tcp" {
			return nil, fmt.Errorf("%w: network: %v", UnexpectedFieldErr, s.Net)
		}
		if s.Type != "none" && s.Type != "" {
			return nil, fmt.Errorf("%w: type: %v", UnexpectedFieldErr, s.Type)
		}
		if s.Aid != "0" && s.Aid != "" {
			return nil, fmt.Errorf("%w: aid: %v, we only support AEAD encryption", UnexpectedFieldErr, s.Aid)
		}

		dialer, err := protocol.NewDialer("vmess", proxy.Direct, protocol.Header{
			ProxyAddress: net.JoinHostPort(s.Add, s.Port),
			Cipher:       "aes-128-gcm",
			Password:     s.ID,
			IsClient:     true,
		})
		return &Dialer{
			Dialer:     dialer,
			supportUDP: true,
			name:       s.Ps,
			link:       link,
		}, nil
	} else if strings.HasPrefix(link, "vless://") {
		//s, err = ParseVlessURL(link)
		return nil, fmt.Errorf("%w: vless", UnexpectedFieldErr)
	} else {
		return nil, InvalidParameterErr
	}
}

func ParseVlessURL(vless string) (data *V2Ray, err error) {
	u, err := url.Parse(vless)
	if err != nil {
		return nil, err
	}
	data = &V2Ray{
		Ps:       u.Fragment,
		Add:      u.Hostname(),
		Port:     u.Port(),
		ID:       u.User.String(),
		Net:      u.Query().Get("type"),
		Type:     u.Query().Get("headerType"),
		SNI:      u.Query().Get("sni"),
		Host:     u.Query().Get("host"),
		Path:     u.Query().Get("path"),
		TLS:      u.Query().Get("security"),
		Flow:     u.Query().Get("flow"),
		Alpn:     u.Query().Get("alpn"),
		Protocol: "vless",
	}
	if data.Net == "" {
		data.Net = "tcp"
	}
	if data.Net == "grpc" {
		data.Path = u.Query().Get("serviceName")
	}
	if data.Type == "" {
		data.Type = "none"
	}
	if data.TLS == "" {
		data.TLS = "none"
	}
	if data.Flow == "" {
		data.Flow = "xtls-rprx-direct"
	}
	if data.Type == "mkcp" || data.Type == "kcp" {
		data.Path = u.Query().Get("seed")
	}
	return data, nil
}

func ParseVmessURL(vmess string) (data *V2Ray, err error) {
	var info V2Ray
	// perform base64 decoding and unmarshal to VmessInfo
	raw, err := common.Base64StdDecode(vmess[8:])
	if err != nil {
		raw, err = common.Base64URLDecode(vmess[8:])
	}
	if err != nil {
		// not in json format, try to resolve as vmess://BASE64(Security:ID@Add:Port)?remarks=Ps&obfsParam=Host&Path=Path&obfs=Net&tls=TLS
		var u *url.URL
		u, err = url.Parse(vmess)
		if err != nil {
			return
		}
		re := regexp.MustCompile(`.*:(.+)@(.+):(\d+)`)
		s := strings.Split(vmess[8:], "?")[0]
		s, err = common.Base64StdDecode(s)
		if err != nil {
			s, err = common.Base64URLDecode(s)
		}
		subMatch := re.FindStringSubmatch(s)
		if subMatch == nil {
			err = fmt.Errorf("unrecognized vmess address")
			return
		}
		q := u.Query()
		ps := q.Get("remarks")
		if ps == "" {
			ps = q.Get("remark")
		}
		obfs := q.Get("obfs")
		obfsParam := q.Get("obfsParam")
		path := q.Get("path")
		if obfs == "kcp" || obfs == "mkcp" {
			m := make(map[string]string)
			//cater to v2rayN definition
			_ = jsoniter.Unmarshal([]byte(obfsParam), &m)
			path = m["seed"]
			obfsParam = ""
		}
		aid := q.Get("alterId")
		if aid == "" {
			aid = q.Get("aid")
		}
		info = V2Ray{
			ID:            subMatch[1],
			Add:           subMatch[2],
			Port:          subMatch[3],
			Ps:            ps,
			Host:          obfsParam,
			Path:          path,
			Net:           obfs,
			Aid:           aid,
			TLS:           map[string]string{"1": "tls"}[q.Get("tls")],
			AllowInsecure: false,
		}
		if info.Net == "websocket" {
			info.Net = "ws"
		}
	} else {
		err = jsoniter.Unmarshal([]byte(raw), &info)
		if err != nil {
			return
		}
	}
	// correct the wrong vmess as much as possible
	if strings.HasPrefix(info.Host, "/") && info.Path == "" {
		info.Path = info.Host
		info.Host = ""
	}
	if info.Aid == "" {
		info.Aid = "0"
	}
	info.Protocol = "vmess"
	return &info, nil
}
