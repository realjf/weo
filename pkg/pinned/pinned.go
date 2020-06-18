package pinned

import (
	"bytes"
	"crypto/sha256"
	"crypto/tls"
	"errors"
	"hash"
	"net"
	"weo/pkg/dialer"
)

type Config struct {
	Hash func() hash.Hash
	Pin []byte
	Config *tls.Config
}

var ErrPinFailure = errors.New("pinned: the peer leaf certificate did not match the provided pin")

func (c *Config) Dial(network, addr string) (net.Conn, error) {
	var conf tls.Config
	if c.Config != nil {
		conf = *c.Config
	}
	conf.InsecureSkipVerify = true

	cn, err := dialer.Retry.Dial(network, addr)
	if err != nil {
		return nil, err
	}

	conn := Conn{
		Conn: tls.Client(cn, &conf),
		Wire: cn,
	}

	if conf.ServerName == "" {
		conf.ServerName, _, _ = net.SplitHostPort(addr)
	}

	if err = conn.Handshake(); err != nil {
		conn.Close()
		return nil, err
	}

	state := conn.ConnectionState()
	hashFunc := c.Hash
	if hashFunc == nil {
		hashFunc = sha256.New
	}
	h := hashFunc()
	h.Write(state.PeerCertificates[0].Raw)
	if !bytes.Equal(h.Sum(nil), c.Pin) {
		conn.Close()
		return nil, ErrPinFailure
	}
	return conn, nil
}

type Conn struct {
	*tls.Conn
	Wire net.Conn
}

func (c Conn) CloseWrite() error {
	if cw, ok := c.Wire.(interface {
		CloseWrite() error
	}); ok {
		return cw.CloseWrite()
	}
	return errors.New("pinned: underlying connection does not support CloseWrite")
}

