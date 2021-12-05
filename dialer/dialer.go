package dialer

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"time"
)

type Dialer struct {
	proxy.Dialer
	supportUDP bool
	name       string
}

func (d *Dialer) SupportUDP() bool {
	return d.supportUDP
}

func (d *Dialer) Name() string {
	return d.name
}

func (d *Dialer) Test(ctx context.Context) (bool, error) {
	cd := ContextDialer{d.Dialer}
	cli := http.Client{
		Transport: &http.Transport{
			DialContext: cd.DialContext,
		},
		Timeout: 5 * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, "GET", "https://gstatic.com/generate_204", nil)
	if err != nil {
		return false, err
	}
	resp, err := cli.Do(req)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr); netErr.Timeout() {
			err = fmt.Errorf("timeout")
		}
		return false, err
	}
	defer resp.Body.Close()
	return resp.StatusCode == 204, nil
}

var (
	UnexpectedFieldErr  = fmt.Errorf("unexpected field")
	InvalidParameterErr = fmt.Errorf("invalid parameters")
)

type FromLinkCreator func(link string) (dialer *Dialer, err error)

var fromLinkCreators = make(map[string]FromLinkCreator)

func FromLinkRegister(name string, creator FromLinkCreator) {
	fromLinkCreators[name] = creator
}

func NewFromLink(name string, link string) (dialer *Dialer, err error) {
	if creator, ok := fromLinkCreators[name]; ok {
		return creator(link)
	} else {
		return nil, fmt.Errorf("unexpected link type: %v", name)
	}
}

type ContextDialer struct {
	Dialer proxy.Dialer
}

func (d *ContextDialer) DialContext(ctx context.Context, network, addr string) (c net.Conn, err error) {
	var done = make(chan struct{})
	go func() {
		c, err = d.Dialer.Dial(network, addr)
		if err != nil {
			return
		}
		select {
		case <-ctx.Done():
			_ = c.Close()
		default:
			close(done)
		}
	}()
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-done:
		return c, err
	}
}
