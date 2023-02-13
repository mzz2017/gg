package dialer

import (
	"context"
	"errors"
	"fmt"
	"golang.org/x/net/proxy"
	"gopkg.in/yaml.v3"
	"net"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

var (
	ConnectivityTestFailedErr = fmt.Errorf("connectivity test failed")
	UnexpectedFieldErr        = fmt.Errorf("unexpected field")
	InvalidParameterErr       = fmt.Errorf("invalid parameters")
)

type Dialer struct {
	proxy.Dialer
	supportUDP bool
	name       string
	protocol   string
	link       string
}

func NewDialer(dialer proxy.Dialer, supportUDP bool, name string, protocol string, link string) *Dialer {
	return &Dialer{
		Dialer:     dialer,
		supportUDP: supportUDP,
		name:       name,
		protocol:   protocol,
		link:       link,
	}
}

func (d *Dialer) SupportUDP() bool {
	return d.supportUDP
}

func (d *Dialer) Name() string {
	return d.name
}

func (d *Dialer) Protocol() string {
	return d.protocol
}

func (d *Dialer) Link() string {
	return d.link
}

func (d *Dialer) Test(ctx context.Context, url string) (bool, error) {
	cd := ContextDialer{d.Dialer}
	cli := http.Client{
		Transport: &http.Transport{
			DialContext: cd.DialContext,
		},
		Timeout: 15 * time.Second,
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, fmt.Errorf("%v: %w", ConnectivityTestFailedErr, err)
	}
	resp, err := cli.Do(req)
	if err != nil {
		var netErr net.Error
		if errors.As(err, &netErr); netErr.Timeout() {
			err = fmt.Errorf("timeout")
		}
		return false, fmt.Errorf("%v: %w", ConnectivityTestFailedErr, err)
	}
	defer resp.Body.Close()
	if page := path.Base(req.URL.Path); strings.HasPrefix(page, "generate_") {
		return strconv.Itoa(resp.StatusCode) == strings.TrimPrefix(page, "generate_"), nil
	}
	return resp.StatusCode >= 200 && resp.StatusCode < 400, nil
}

type FromLinkCreator func(link string, opt *GlobalOption) (dialer *Dialer, err error)

var fromLinkCreators = make(map[string]FromLinkCreator)

func FromLinkRegister(name string, creator FromLinkCreator) {
	fromLinkCreators[name] = creator
}

func NewFromLink(name string, link string, opt *GlobalOption) (dialer *Dialer, err error) {
	if creator, ok := fromLinkCreators[name]; ok {
		return creator(link, opt)
	} else {
		return nil, fmt.Errorf("unexpected link type: %v", name)
	}
}

type FromClashCreator func(clashObj *yaml.Node, opt *GlobalOption) (dialer *Dialer, err error)

var fromClashCreators = make(map[string]FromClashCreator)

func FromClashRegister(name string, creator FromClashCreator) {
	fromClashCreators[name] = creator
}

func NewFromClash(clashObj *yaml.Node, opt *GlobalOption) (dialer *Dialer, err error) {
	preUnload := make(map[string]interface{})
	if err := clashObj.Decode(&preUnload); err != nil {
		return nil, err
	}
	name, _ := preUnload["type"].(string)
	if creator, ok := fromClashCreators[name]; ok {
		return creator(clashObj, opt)
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
			close(done)
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

type GlobalOption struct {
	AllowInsecure bool
}
