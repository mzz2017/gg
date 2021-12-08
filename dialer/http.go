package dialer

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/protocol/http"
	"net/url"
)

func init() {
	FromLinkRegister("http", NewHTTP)
	FromLinkRegister("https", NewHTTP)
}

type HTTP struct {
	Name     string `json:"name"`
	Server   string `json:"server"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Protocol string `json:"protocol"`
}

func NewHTTP(link string) (*Dialer, error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, InvalidParameterErr
	}

	dialer, err := http.NewHTTPProxy(u, SymmetricDirect)
	if err != nil {
		return nil, err
	}
	return &Dialer{
		Dialer:     dialer,
		supportUDP: false,
		name:       u.Fragment,
		link:       link,
	}, nil
}