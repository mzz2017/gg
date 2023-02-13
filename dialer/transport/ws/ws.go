package ws

import (
	"crypto/tls"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/mzz2017/gg/common"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"net/url"
)

// Ws is a base Ws struct
type Ws struct {
	dialer   proxy.Dialer
	wsAddr   string
	header   http.Header
	wsDialer *websocket.Dialer
}

// NewWs returns a Ws infra.
func NewWs(s string, d proxy.Dialer) (*Ws, error) {
	u, err := url.Parse(s)
	if err != nil {
		return nil, fmt.Errorf("NewWs: %w", err)
	}

	t := &Ws{
		dialer: d,
	}

	query := u.Query()
	host := query.Get("host")
	if host == "" {
		host = u.Hostname()
	}
	t.header = http.Header{}
	t.header.Set("Host", host)

	wsUrl := url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
	}
	t.wsAddr = wsUrl.String() + u.Path
	t.wsDialer = &websocket.Dialer{
		NetDial: d.Dial,
		//Subprotocols: []string{"binary"},
	}
	if u.Scheme == "wss" {
		skipVerify := common.StringToBool(query.Get("allowInsecure")) ||
			common.StringToBool(query.Get("skipVerify"))
		t.wsDialer.TLSClientConfig = &tls.Config{
			ServerName:         u.Query().Get("sni"),
			InsecureSkipVerify: skipVerify,
		}
	}
	return t, nil
}

// Dial connects to the address addr on the network net via the infra.
func (s *Ws) Dial(network, addr string) (net.Conn, error) {
	rc, _, err := s.wsDialer.Dial(s.wsAddr, s.header)
	if err != nil {
		return nil, fmt.Errorf("[Ws]: dial to %s: %w", s.wsAddr, err)
	}
	return newConn(rc), err
}
