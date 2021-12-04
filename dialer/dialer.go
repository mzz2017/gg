package dialer

import (
	"fmt"
	"golang.org/x/net/proxy"
)

type Dialer struct {
	proxy.Dialer
	supportUDP bool
}

func (d *Dialer) SupportUDP() bool {
	return d.supportUDP
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
