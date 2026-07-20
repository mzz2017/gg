package shadowsocks

import (
	"encoding/base64"
	"testing"
)

func TestSS2022DialerFromURL(t *testing.T) {
	key := base64.StdEncoding.EncodeToString(make([]byte, 16))
	link := (&Shadowsocks{
		Cipher:   "2022-blake3-aes-128-gcm",
		Password: key,
		Server:   "127.0.0.1",
		Port:     8388,
		Name:     "ss2022",
		UDP:      true,
		Protocol: "shadowsocks",
	}).ExportToURL()
	parsed, err := ParseSSURL(link)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Cipher != "2022-blake3-aes-128-gcm" {
		t.Fatalf("unexpected cipher: %v", parsed.Cipher)
	}
	if parsed.Password != key {
		t.Fatalf("unexpected password: %v", parsed.Password)
	}
	if _, err = parsed.Dialer(); err != nil {
		t.Fatal(err)
	}
}

func TestSS2022DialerRejectsBadKeyLength(t *testing.T) {
	s := &Shadowsocks{
		Cipher:   "2022-blake3-aes-128-gcm",
		Password: base64.StdEncoding.EncodeToString(make([]byte, 15)),
		Server:   "127.0.0.1",
		Port:     8388,
		UDP:      true,
		Protocol: "shadowsocks",
	}
	if _, err := s.Dialer(); err == nil {
		t.Fatal("expected bad key length error")
	}
}

func TestLegacyShadowsocksDialerUsesOutbound(t *testing.T) {
	s := &Shadowsocks{
		Cipher:   "aes-128-gcm",
		Password: "password",
		Server:   "192.0.2.1",
		Port:     8388,
		UDP:      true,
		Protocol: "shadowsocks",
	}
	if _, err := s.Dialer(); err != nil {
		t.Fatal(err)
	}
}
