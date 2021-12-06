package cmd

import (
	"github.com/mzz2017/gg/common"
	"github.com/mzz2017/gg/dialer"
	"github.com/sirupsen/logrus"
	"io"
	"net/http"
	"strings"
)

func getDialersFromSubscription(log *logrus.Logger, subscription string) (dialers []*dialer.Dialer, err error) {
	resp, err := http.Get(subscription)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	// base64 decode
	raw, err := common.Base64StdDecode(string(b))
	if err != nil {
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
			log.Tracef("%v: %v", e, line)
			continue
		}
		dialers = append(dialers, d)
	}
	return dialers, nil
}
