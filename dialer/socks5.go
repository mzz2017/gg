package dialer

import (
	"github.com/txthinking/socks5"
	"net/url"
)

func init() {
	FromLinkRegister("socks", NewSocks5)
	FromLinkRegister("socks5", NewSocks5)
}

type Socks5 struct {
	Name     string `json:"name"`
	Server   string `json:"server"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Protocol string `json:"protocol"`
}

func NewSocks5(link string) (*Dialer, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, InvalidParameterErr
	}
	pwd, _ := u.User.Password()
	dialer, err := socks5.NewClient(u.Host, u.User.Username(), pwd, 0, 0)
	if err != nil {
		return nil, err
	}
	return &Dialer{
		Dialer:     dialer,
		supportUDP: true,
		name:       u.Fragment,
		link:       link,
	}, nil
}
