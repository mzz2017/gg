package dialer
//
//import (
//	"net"
//	"net/url"
//	"strconv"
//)
//
//func init() {
//	//FromLinkRegister("http", NewHTTP)
//	//FromLinkRegister("https", NewHTTP)
//	//FromLinkRegister("http-proxy", NewHTTP)
//	//FromLinkRegister("https-proxy", NewHTTP)
//}
//
//type HTTP struct {
//	Name     string `json:"name"`
//	Server   string `json:"server"`
//	Port     int    `json:"port"`
//	Username string `json:"username"`
//	Password string `json:"password"`
//	Protocol string `json:"protocol"`
//}
//
//func NewHTTP(link string) (ServerObj, error) {
//	return ParseHttpURL(link)
//}
//
//func ParseHttpURL(u string) (data *HTTP, err error) {
//	t, err := url.Parse(u)
//	if err != nil {
//		return nil, InvalidParameterErr
//	}
//	port, err := strconv.Atoi(t.Port())
//	if err != nil {
//		return nil, InvalidParameterErr
//	}
//	data = &HTTP{
//		Name:   t.Fragment,
//		Server: t.Hostname(),
//		Port:   port,
//	}
//	if t.User != nil && len(t.User.String()) > 0 {
//		data.Username = t.User.Username()
//		data.Password, _ = t.User.Password()
//	}
//	switch t.Scheme {
//	case "https-proxy", "https":
//		data.Protocol = "https"
//		if data.Port == 0 {
//			data.Port = 443
//		}
//	case "http-proxy", "http":
//		data.Protocol = "http"
//		if data.Port == 0 {
//			data.Port = 80
//		}
//	default:
//		data.Protocol = t.Scheme
//	}
//	return data, nil
//}
//
//
//func (h *HTTP) ExportToURL() string {
//	var user *url.Userinfo
//	if h.Username != "" && h.Password != "" {
//		user = url.UserPassword(h.Username, h.Password)
//	}
//	u := &url.URL{
//		Scheme:   h.Protocol,
//		User:     user,
//		Host:     net.JoinHostPort(h.Server, strconv.Itoa(h.Port)),
//		Fragment: h.Name,
//	}
//	return u.String()
//}
//
//func (h *HTTP) NeedPlugin() bool {
//	return false
//}
//
//func (h *HTTP) ProtoToShow() string {
//	return h.Protocol
//}
//
//func (h *HTTP) GetProtocol() string {
//	return h.Protocol
//}
//
//func (h *HTTP) GetHostname() string {
//	return h.Server
//}
//
//func (h *HTTP) GetPort() int {
//	return h.Port
//}
//
//func (h *HTTP) GetName() string {
//	return h.Name
//}
//
//func (h *HTTP) SetName(name string) {
//	h.Name = name
//}
