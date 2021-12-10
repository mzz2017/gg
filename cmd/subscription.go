package cmd

import (
	"bytes"
	"github.com/mzz2017/gg/common"
	"github.com/mzz2017/gg/dialer"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"strings"
)

type ClashConfig struct {
	Proxy []yaml.Node `yaml:"proxies"`
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
	if dialers, err = resolveSubscriptionAsClash(log, b); err == nil {
		return dialers, nil
	}
	return resolveSubscriptionAsBase64(log, b), nil
}
