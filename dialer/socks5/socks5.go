package socks5

import (
	"fmt"
	"github.com/mzz2017/gg/dialer"
	"github.com/txthinking/socks5"
	"gopkg.in/yaml.v3"
	"net"
	"net/url"
	"strconv"
)

func init() {
	dialer.FromLinkRegister("socks", NewSocks5)
	dialer.FromLinkRegister("socks5", NewSocks5)
	dialer.FromClashRegister("socks5", NewSocks5FromClashObj)
}

type Socks5 struct {
	Name     string `json:"name"`
	Server   string `json:"server"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Protocol string `json:"protocol"`
	UDP      bool   `json:"udp"`
}

func NewSocks5(link string) (*dialer.Dialer, error) {
	s, err := ParseSocks5URL(link)
	if err != nil {
		return nil, dialer.InvalidParameterErr
	}
	return s.Dialer()
}

func NewSocks5FromClashObj(o *yaml.Node) (*dialer.Dialer, error) {
	s, err := ParseClash(o)
	if err != nil {
		return nil, err
	}
	return s.Dialer()
}

func (s *Socks5) Dialer() (*dialer.Dialer, error) {
	d, err := socks5.NewClient(net.JoinHostPort(s.Server, strconv.Itoa(s.Port)), s.Username, s.Password, 0, 0)
	if err != nil {
		return nil, err
	}
	return dialer.NewDialer(d, s.UDP, s.Name, s.ExportToURL()), nil
}

func ParseClash(o *yaml.Node) (data *Socks5, err error) {
	type Socks5Option struct {
		Name           string `yaml:"name"`
		Server         string `yaml:"server"`
		Port           int    `yaml:"port"`
		UserName       string `yaml:"username,omitempty"`
		Password       string `yaml:"password,omitempty"`
		TLS            bool   `yaml:"tls,omitempty"`
		UDP            bool   `yaml:"udp,omitempty"`
		SkipCertVerify bool   `yaml:"skip-cert-verify,omitempty"`
	}
	var option Socks5Option
	if err = o.Decode(&option); err != nil {
		return nil, err
	}
	if option.TLS {
		return nil, fmt.Errorf("%w: tls=true", dialer.UnexpectedFieldErr)
	}
	if option.SkipCertVerify {
		return nil, fmt.Errorf("%w: skip-cert-verify=true", dialer.UnexpectedFieldErr)
	}
	return &Socks5{
		Name:     option.Name,
		Server:   option.Server,
		Port:     option.Port,
		Username: option.UserName,
		Password: option.Password,
		UDP:      option.UDP,
		Protocol: "socks5",
	}, nil
}

func ParseSocks5URL(link string) (data *Socks5, err error) {
	u, err := url.Parse(link)
	if err != nil {
		return nil, dialer.InvalidParameterErr
	}
	pwd, _ := u.User.Password()
	strPort := u.Port()
	port, err := strconv.Atoi(strPort)
	if err != nil {
		return nil, err
	}
	return &Socks5{
		Name:     u.Fragment,
		Server:   u.Hostname(),
		Port:     port,
		Username: u.User.Username(),
		Password: pwd,
		UDP:      true,
		Protocol: "socks5",
	}, nil
}

func (s *Socks5) ExportToURL() string {
	var user *url.Userinfo
	if s.Password != "" {
		user = url.UserPassword(s.Username, s.Password)
	} else {
		user = url.User(s.Username)
	}
	u := url.URL{
		Scheme:   "socks5",
		User:     user,
		Host:     net.JoinHostPort(s.Server, strconv.Itoa(s.Port)),
		Fragment: s.Name,
	}
	return u.String()
}
