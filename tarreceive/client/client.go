package client

import (
	"errors"
	"github.com/flynn/flynn/pkg/httphelper"
	"weo/pkg/httpclient"
)

var ErrNotFound = errors.New("layer not found")


type Config struct {
	Pin   []byte
	Domain string
}

type Client struct {
	*httpclient.Client
}

func NewClient(url, key string) *Client {
	return newClient(url, key, httphelper.RetryClient)
}

