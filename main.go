package main

import (
	"github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/protocol"
	_ "github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/protocol/shadowsocks"
	_ "github.com/e14914c0-6759-480d-be89-66b7b7676451/BitterJohn/protocol/vmess"
	"github.com/mzz2017/gg/tracer"
	"github.com/sirupsen/logrus"
	"golang.org/x/net/proxy"
	"os"
	"os/exec"
)

func main() {
	logrus.SetLevel(logrus.TraceLevel)
	dialer, err := protocol.NewDialer("vmess", proxy.Direct, protocol.Header{
		ProxyAddress: "localhost:27311",
		Cipher:       "aes-128-gcm",
		Password:     "333ef100-a0f3-5423-92dc-65fae4c0a45f",
		IsClient:     true,
	})
	if err != nil {
		panic(err)
	}
	log := logrus.New()
	log.SetLevel(logrus.TraceLevel)
	p, err := exec.LookPath(os.Args[1])
	if err != nil {
		panic(err)
	}
	t, err := tracer.New(
		p,
		os.Args[1:],
		&os.ProcAttr{Files: []*os.File{os.Stdin, os.Stdout, os.Stderr}},
		dialer,
		false,
		log,
	)
	if err != nil {
		panic(err)
	}
	code, err := t.Wait()
	if err != nil {
		panic(err)
	}
	os.Exit(code)
}
