package dialer

import (
	"fmt"
	"github.com/mzz2017/gg/common"
	ssr "github.com/v2rayA/shadowsocksR/client"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
)

func init() {
	FromLinkRegister("shadowsocksr", NewShadowsocksR)
	FromLinkRegister("ssr", NewShadowsocksR)
}

type ShadowsocksR struct {
	Name       string `json:"name"`
	Server     string `json:"server"`
	Port       int    `json:"port"`
	Password   string `json:"password"`
	Cipher     string `json:"cipher"`
	Proto      string `json:"proto"`
	ProtoParam string `json:"protoParam"`
	Obfs       string `json:"obfs"`
	ObfsParam  string `json:"obfsParam"`
	Protocol   string `json:"protocol"`
}

func NewShadowsocksR(link string) (*Dialer, error) {
	s, err := ParseSSRURL(link)
	if err != nil {
		return nil, err
	}
	log.Println(s)
	u := url.URL{
		Scheme: "ssr",
		User:   url.UserPassword(s.Cipher, s.Password),
		Host:   net.JoinHostPort(s.Server, strconv.Itoa(s.Port)),
		RawQuery: url.Values{
			"protocol":       []string{s.Proto},
			"protocol_param": []string{s.ProtoParam},
			"obfs":           []string{s.Obfs},
			"obfs_param":     []string{s.ObfsParam},
		}.Encode(),
	}
	dialer := FullconeDirect
	dialer, err = ssr.NewSSR(u.String(), dialer, nil)
	return &Dialer{
		Dialer:     dialer,
		supportUDP: false,
		name:       s.Name,
		link:       link,
	}, nil
}

func ParseSSRURL(u string) (data *ShadowsocksR, err error) {
	// parse attempts to parse ss:// links
	parse := func(content string) (v ShadowsocksR, ok bool) {
		arr := strings.Split(content, "/?")
		if strings.Contains(content, ":") && len(arr) < 2 {
			content += "/?remarks=&protoparam=&obfsparam="
			arr = strings.Split(content, "/?")
		} else if len(arr) != 2 {
			return v, false
		}
		pre := strings.Split(arr[0], ":")
		if len(pre) > 6 {
			//if the length is more than 6, it means that the host contains the characters:,
			//re-merge the first few groups into the host
			pre[len(pre)-6] = strings.Join(pre[:len(pre)-5], ":")
			pre = pre[len(pre)-6:]
		} else if len(pre) < 6 {
			return v, false
		}
		q, err := url.ParseQuery(arr[1])
		if err != nil {
			return v, false
		}
		pswd, _ := common.Base64URLDecode(pre[5])
		add, _ := common.Base64URLDecode(pre[0])
		remarks, _ := common.Base64URLDecode(q.Get("remarks"))
		protoparam, _ := common.Base64URLDecode(q.Get("protoparam"))
		obfsparam, _ := common.Base64URLDecode(q.Get("obfsparam"))
		port, err := strconv.Atoi(pre[1])
		if err != nil {
			return v, false
		}
		v = ShadowsocksR{
			Name:       remarks,
			Server:     add,
			Port:       port,
			Password:   pswd,
			Cipher:     pre[3],
			Proto:      pre[2],
			ProtoParam: protoparam,
			Obfs:       pre[4],
			ObfsParam:  obfsparam,
			Protocol:   "shadowsocksr",
		}
		return v, true
	}
	content := u[6:]
	var (
		info ShadowsocksR
		ok   bool
	)
	// try parsing the ssr:// link, if it fails, base64 decode first
	if info, ok = parse(content); !ok {
		// perform base64 decoding and parse again
		content, err = common.Base64StdDecode(content)
		if err != nil {
			content, err = common.Base64URLDecode(content)
			if err != nil {
				return
			}
		}
		info, ok = parse(content)
	}
	if !ok {
		err = fmt.Errorf("%w: unrecognized ssr address", InvalidParameterErr)
		return
	}
	return &info, nil
}
