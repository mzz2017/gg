package main

import (
	"github.com/json-iterator/go/extra"
	"github.com/mzz2017/gg/cmd"
	"net/http"
	"os"
	"time"

	_ "github.com/mzz2017/gg/dialer/http"
	_ "github.com/mzz2017/gg/dialer/shadowsocks"
	_ "github.com/mzz2017/gg/dialer/shadowsocksr"
	_ "github.com/mzz2017/gg/dialer/socks"
	_ "github.com/mzz2017/gg/dialer/trojan"
	_ "github.com/mzz2017/gg/dialer/v2ray"
	_ "github.com/mzz2017/softwind/protocol/shadowsocks"
	_ "github.com/mzz2017/softwind/protocol/trojanc"
	_ "github.com/mzz2017/softwind/protocol/vless"
	_ "github.com/mzz2017/softwind/protocol/vmess"
)

func main() {
	extra.RegisterFuzzyDecoders()

	http.DefaultClient.Timeout = 30 * time.Second
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
