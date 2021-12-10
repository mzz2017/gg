package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mzz2017/gg/common"
	"github.com/mzz2017/gg/dialer"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

type ClashConfig struct {
	Proxy []yaml.Node `yaml:"proxies"`
}

type SIP008 struct {
	Version        int            `json:"version"`
	Servers        []SIP008Server `json:"servers"`
	BytesUsed      int64          `json:"bytes_used"`
	BytesRemaining int64          `json:"bytes_remaining"`
}

type SIP008Server struct {
	Id         string `json:"id"`
	Remarks    string `json:"remarks"`
	Server     string `json:"server"`
	ServerPort int    `json:"server_port"`
	Password   string `json:"password"`
	Method     string `json:"method"`
	Plugin     string `json:"plugin"`
	PluginOpts string `json:"plugin_opts"`
}

func resolveSubscriptionAsClash(log *logrus.Logger, b []byte) (dialers []*dialer.Dialer, err error) {
	log.Traceln("try to resolve as Clash")

	var conf ClashConfig
	if err = yaml.NewDecoder(bytes.NewReader(b)).Decode(&conf); err != nil {
		return nil, err
	}
	for i, node := range conf.Proxy {
		d, e := dialer.NewFromClash(&node)
		if e != nil {
			log.Tracef("proxies[%v]: %v\n", i, e)
			continue
		}
		dialers = append(dialers, d)
	}
	return dialers, nil
}

func resolveSubscriptionAsBase64(log *logrus.Logger, b []byte) (dialers []*dialer.Dialer) {
	log.Traceln("try to resolve as base64")

	// base64 decode
	raw, e := common.Base64StdDecode(string(b))
	if e != nil {
		raw, _ = common.Base64URLDecode(string(b))
	}
	lines := strings.Split(raw, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		d, e := GetDialerFromLink(line, false)
		if e != nil {
			log.Tracef("%v: %v\n", e, line)
			continue
		}
		dialers = append(dialers, d)
	}
	return dialers
}

func resolveSubscriptionAsSIP008(log *logrus.Logger, b []byte) (dialers []*dialer.Dialer, err error) {
	log.Traceln("try to resolve as SIP008")

	var sip SIP008
	err = json.Unmarshal(b, &sip)
	if err != nil {
		return
	}
	if sip.Version != 1 || sip.Servers == nil {
		return nil, fmt.Errorf("does not seems like a SIP008 subscription")
	}
	for i, server := range sip.Servers {
		u := url.URL{
			Scheme:   "ss",
			User:     url.UserPassword(server.Method, server.Password),
			Host:     net.JoinHostPort(server.Server, strconv.Itoa(server.ServerPort)),
			RawQuery: url.Values{"plugin": []string{server.PluginOpts}}.Encode(),
			Fragment: server.Remarks,
		}
		d, e := dialer.NewFromLink("shadowsocks", u.String())
		if e != nil {
			log.Tracef("servers[%v]: %v\n", i, e)
			continue
		}
		dialers = append(dialers, d)
	}
	return
}

func pullDialersFromSubscription(log *logrus.Logger, subscription string) (dialers []*dialer.Dialer, err error) {
	resp, err := http.Get(subscription)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if dialers, err = resolveSubscriptionAsSIP008(log, b); err == nil {
		return dialers, nil
	} else {
		log.Traceln(err)
	}
	if dialers, err = resolveSubscriptionAsClash(log, b); err == nil {
		return dialers, nil
	} else {
		log.Traceln(err)
	}
	return resolveSubscriptionAsBase64(log, b), nil
}
