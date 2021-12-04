package main

import (
	_ "github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/protocol/shadowsocks"
	_ "github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/protocol/vmess"
	"github.com/mzz2017/gg/cmd"
	"os"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
