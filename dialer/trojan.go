package dialer
//
//import (
//	"fmt"
//	"net"
//	"net/url"
//	"strconv"
//	"strings"
//)
//
//func init() {
//	FromLinkRegister("trojan", NewTrojan)
//	FromLinkRegister("trojan-go", NewTrojan)
//	EmptyRegister("trojan", func() (ServerObj, error) {
//		return new(Trojan), nil
//	})
//	EmptyRegister("trojan-go", func() (ServerObj, error) {
//		return new(Trojan), nil
//	})
//}
//
//type Trojan struct {
//	Name          string `json:"name"`
//	Server        string `json:"server"`
//	Port          int    `json:"port"`
//	Password      string `json:"password"`
//	Sni           string `json:"sni"`
//	Type          string `json:"type"`
//	Encryption    string `json:"encryption"`
//	Host          string `json:"host"`
//	Path          string `json:"path"`
//	AllowInsecure bool   `json:"allowInsecure"`
//	Protocol      string `json:"protocol"`
//}
//
//func NewTrojan(link string) (ServerObj, error) {
//	return ParseTrojanURL(link)
//}
//
//func ParseTrojanURL(u string) (data *Trojan, err error) {
//	//trojan://password@server:port#escape(remarks)
//	t, err := url.Parse(u)
//	if err != nil {
//		err = fmt.Errorf("invalid trojan format")
//		return
//	}
//	allowInsecure := t.Query().Get("allowInsecure")
//	sni := t.Query().Get("peer")
//	if sni == "" {
//		sni = t.Query().Get("sni")
//	}
//	if sni == "" {
//		sni = t.Hostname()
//	}
//	port, err := strconv.Atoi(t.Port())
//	if err != nil {
//		return nil, InvalidParameterErr
//	}
//	data = &Trojan{
//		Name:          t.Fragment,
//		Server:        t.Hostname(),
//		Port:          port,
//		Password:      t.User.Username(),
//		Sni:           sni,
//		AllowInsecure: allowInsecure == "1" || allowInsecure == "true",
//		Protocol:      "trojan",
//	}
//	if t.Scheme == "trojan-go" {
//		data.Protocol = "trojan-go"
//		data.Encryption = t.Query().Get("encryption")
//		data.Host = t.Query().Get("host")
//		data.Path = t.Query().Get("path")
//		data.Type = t.Query().Get("type")
//		data.AllowInsecure = false
//	}
//	return data, nil
//}
//
//func (t *Trojan) ExportToURL() string {
//	u := &url.URL{
//		Scheme:   "trojan",
//		User:     url.User(t.Password),
//		Host:     net.JoinHostPort(t.Server, strconv.Itoa(t.Port)),
//		Fragment: t.Name,
//	}
//	q := u.Query()
//	if t.AllowInsecure {
//		q.Set("allowInsecure", "1")
//	}
//	setValue(&q, "sni", t.Sni)
//
//	if t.Protocol == "trojan-go" {
//		u.Scheme = "trojan-go"
//		setValue(&q, "host", t.Host)
//		setValue(&q, "encryption", t.Encryption)
//		setValue(&q, "type", t.Type)
//		setValue(&q, "path", t.Path)
//	}
//	u.RawQuery = q.Encode()
//	return u.String()
//}
//
//func (t *Trojan) NeedPlugin() bool {
//	return t.Protocol == "trojan-go"
//}
//
//func (t *Trojan) ProtoToShow() string {
//	if t.Protocol == "trojan" {
//		return t.Protocol
//	}
//	if t.Encryption == "" {
//		return fmt.Sprintf("%v(%v)", t.Protocol, t.Type)
//	}
//	return fmt.Sprintf("%v(%v+%v)", t.Protocol, t.Type, strings.Split(t.Encryption, ";")[0])
//}
//
//func (t *Trojan) GetProtocol() string {
//	return t.Protocol
//}
//
//func (t *Trojan) GetHostname() string {
//	return t.Server
//}
//
//func (t *Trojan) GetPort() int {
//	return t.Port
//}
//
//func (t *Trojan) GetName() string {
//	return t.Name
//}
//
//func (t *Trojan) SetName(name string) {
//	t.Name = name
//}
