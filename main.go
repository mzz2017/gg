package main

import (
	"github.com/json-iterator/go/extra"
	"github.com/mzz2017/gg/cmd"
	"net/http"
	"os"
	"time"

	_ "github.com/daeuniverse/outbound/protocol/shadowsocks"
	_ "github.com/daeuniverse/outbound/protocol/trojanc"
	_ "github.com/daeuniverse/outbound/protocol/vless"
	_ "github.com/daeuniverse/outbound/protocol/vmess"
	_ "github.com/mzz2017/gg/dialer/anytls"
	_ "github.com/mzz2017/gg/dialer/http"
	_ "github.com/mzz2017/gg/dialer/hysteria2"
	_ "github.com/mzz2017/gg/dialer/shadowsocks"
	_ "github.com/mzz2017/gg/dialer/shadowsocksr"
	_ "github.com/mzz2017/gg/dialer/socks"
	_ "github.com/mzz2017/gg/dialer/trojan"
	_ "github.com/mzz2017/gg/dialer/v2ray"
)

func main() {
	extra.RegisterFuzzyDecoders()

	http.DefaultClient.Timeout = 30 * time.Second
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
