package grpc

import (
	"bytes"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"sync"
	"time"

	"ekyu.moe/leb128"
	"encoding/binary"
	"golang.org/x/net/http2"
)

type GunConn struct {
	reader io.Reader
	writer io.Writer
	closer io.Closer
	local  net.Addr
	remote net.Addr
	// mu protect done
	mu   sync.Mutex
	done chan struct{}

	toRead []byte
	readAt int
}

type Client struct {
	client  *http.Client
	url     *url.URL
	headers http.Header
}

type Config struct {
	RemoteAddr  string
	ServerName  string
	ServiceName string
	Cleartext   bool
	tlsConfig   *tls.Config
}

func NewGunClient(config *Config) *Client {
	var dialFunc func(network, addr string, cfg *tls.Config) (net.Conn, error) = nil
	if config.Cleartext {
		dialFunc = func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			return net.Dial(network, addr)
		}
	} else {
		dialFunc = func(network, addr string, cfg *tls.Config) (net.Conn, error) {
			pconn, err := net.Dial(network, addr)
			if err != nil {
				return nil, err
			}

			cn := tls.Client(pconn, cfg)
			if err := cn.Handshake(); err != nil {
				return nil, err
			}
			state := cn.ConnectionState()
			if p := state.NegotiatedProtocol; p != http2.NextProtoTLS {
				return nil, errors.New("http2: unexpected ALPN protocol " + p + "; want q" + http2.NextProtoTLS)
			}
			return cn, nil
		}
	}

	if config.tlsConfig == nil && config.ServerName != "" {
		config.tlsConfig = new(tls.Config)
		config.tlsConfig.ServerName = config.ServerName
		config.tlsConfig.NextProtos = []string{"h2"}
	}

	client := &http.Client{
		Transport: &http2.Transport{
			DialTLS:            dialFunc,
			TLSClientConfig:    config.tlsConfig,
			AllowHTTP:          false,
			DisableCompression: true,
			ReadIdleTimeout:    0,
			PingTimeout:        0,
		},
	}

	var serviceName = "GunService"
	if config.ServiceName != "" {
		serviceName = config.ServiceName
	}

	return &Client{
		client: client,
		url: &url.URL{
			Scheme: "https",
			Host:   config.RemoteAddr,
			Path:   fmt.Sprintf("/%s/Tun", serviceName),
		},
		headers: http.Header{
			"content-type": []string{"application/grpc"},
			"user-agent":   []string{"grpc-go/1.36.0"},
			"te":           []string{"trailers"},
		},
	}
}

type ChainedClosable []io.Closer

// Close implements io.Closer.Close().
func (cc ChainedClosable) Close() error {
	for _, c := range cc {
		_ = c.Close()
	}
	return nil
}

func (cli *Client) DialConn() (net.Conn, error) {
	reader, writer := io.Pipe()
	request := &http.Request{
		Method:     http.MethodPost,
		Body:       reader,
		URL:        cli.url,
		Proto:      "HTTP/2",
		ProtoMajor: 2,
		ProtoMinor: 0,
		Header:     cli.headers,
	}
	anotherReader, anotherWriter := io.Pipe()
	go func() {
		defer anotherWriter.Close()
		response, err := cli.client.Do(request)
		if err != nil {
			return
		}
		_, _ = io.Copy(anotherWriter, response.Body)
	}()

	return newGunConn(anotherReader, writer, ChainedClosable{reader, writer, anotherReader}, nil, nil), nil
}

var (
	ErrInvalidLength = errors.New("invalid length")
)

func newGunConn(reader io.Reader, writer io.Writer, closer io.Closer, local net.Addr, remote net.Addr) *GunConn {
	if local == nil {
		local = &net.TCPAddr{
			IP:   []byte{0, 0, 0, 0},
			Port: 0,
		}
	}
	if remote == nil {
		remote = &net.TCPAddr{
			IP:   []byte{0, 0, 0, 0},
			Port: 0,
		}
	}
	return &GunConn{
		reader: reader,
		writer: writer,
		closer: closer,
		local:  local,
		remote: remote,
		done:   make(chan struct{}),
	}
}

func (g *GunConn) isClosed() bool {
	select {
	case <-g.done:
		return true
	default:
		return false
	}
}

func (g *GunConn) Read(b []byte) (n int, err error) {
	if g.toRead != nil {
		n = copy(b, g.toRead[g.readAt:])
		g.readAt += n
		if g.readAt >= len(g.toRead) {
			g.toRead = nil
		}
		return n, nil
	}
	buf := make([]byte, 5)
	n, err = io.ReadFull(g.reader, buf)
	if err != nil {
		return 0, err
	}
	//log.Printf("GRPC Header: %x", buf[:n])
	grpcPayloadLen := binary.BigEndian.Uint32(buf[1:])
	//log.Printf("GRPC Payload Length: %d", grpcPayloadLen)

	buf = make([]byte, grpcPayloadLen)
	n, err = io.ReadFull(g.reader, buf)
	if err != nil {
		return 0, io.ErrUnexpectedEOF
	}
	protobufPayloadLen, protobufLengthLen := leb128.DecodeUleb128(buf[1:])
	//log.Printf("Protobuf Payload Length: %d, Length Len: %d", protobufPayloadLen, protobufLengthLen)
	if protobufLengthLen == 0 {
		return 0, ErrInvalidLength
	}
	if grpcPayloadLen != uint32(protobufPayloadLen)+uint32(protobufLengthLen)+1 {
		return 0, ErrInvalidLength
	}
	n = copy(b, buf[1+protobufLengthLen:])
	if n < int(protobufPayloadLen) {
		g.toRead = buf
		g.readAt = 1 + int(protobufLengthLen) + n
	}
	return n, nil
}

func (g *GunConn) Write(b []byte) (n int, err error) {
	if g.isClosed() {
		return 0, io.ErrClosedPipe
	}
	protobufHeader := leb128.AppendUleb128([]byte{0x0A}, uint64(len(b)))
	grpcHeader := make([]byte, 5)
	grpcPayloadLen := uint32(len(protobufHeader) + len(b))
	binary.BigEndian.PutUint32(grpcHeader[1:5], grpcPayloadLen)
	_, err = io.Copy(g.writer, io.MultiReader(bytes.NewReader(grpcHeader), bytes.NewReader(protobufHeader), bytes.NewReader(b)))
	if f, ok := g.writer.(http.Flusher); ok {
		f.Flush()
	}
	return len(b), err
}

func (g *GunConn) Close() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	select {
	case <-g.done:
		return nil
	default:
		close(g.done)
		return g.closer.Close()
	}
}

func (g *GunConn) LocalAddr() net.Addr {
	return g.local
}

func (g *GunConn) RemoteAddr() net.Addr {
	return g.remote
}

func (g *GunConn) SetDeadline(t time.Time) error {
	return nil
}

func (g *GunConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (g *GunConn) SetWriteDeadline(t time.Time) error {
	return nil
}
