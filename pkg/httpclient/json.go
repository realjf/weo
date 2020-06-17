package httpclient

import (
	"bytes"
	"encoding/json"
	"io"
	"net"
	"net/http"
)

type DialFunc func(network, addr string) (net.Conn, error)

type writeCloser interface {
	CloseWrite() error
}


type writerCloser interface {
	io.WriteCloser
	CloseWrite() error
}

type ReadWriteCloser interface {
	io.ReadWriteCloser
	CloseWrite() error
}

type Client struct {
	ErrNotFound error
	URL 		string
	Key			string
	Host		string
	HTTP 		*http.Client
	HijackDial	DialFunc
}

func ToJSON(v interface{}) (io.Reader, error) {
	data, err := json.Marshal(v)
	return bytes.NewBuffer(data), err
}

func (c *Client) prepareReq(method, rawurl string, header http.Header, in interface{}) (*http.Request, error) {
	var payload io.Reader
	switch v := in.(type) {
	case io.Reader:
		payload = v
	case nil:
	default:
		var err error
		payload, err = ToJSON(in)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequest(method, rawurl, payload)
	if err != nil {
		return nil, err
	}
	if header == nil {
		header = make(http.Header)
	}
	if header.Get("Content-Type") == "" {
		header.Set("Content-Type", "application/json")
	}
	req.Header = header
	if c.Key != "" {
		req.SetBasicAuth("", c.Key)
	}
	if c.Host != "" {
		req.Host = c.Host
	}

	return req, nil
}





