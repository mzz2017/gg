package http

import "testing"

func TestDialerUsesOutbound(t *testing.T) {
	h := &HTTP{
		Server:   "192.0.2.1",
		Port:     8080,
		Protocol: "http",
	}
	if _, err := h.Dialer(); err != nil {
		t.Fatal(err)
	}
}
