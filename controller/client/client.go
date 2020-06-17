package controller

import (
	"github.com/flynn/flynn/pkg/stream"
	"net/http"
	"time"
	v1controller "weo/controller/client/v1"
	"weo/pkg/httpclient"
)

// client用于处理cli的请求
type Client interface {
	SetKey(newKey string)
	GetCACert() ([]byte, error)
	StreamFormations(since *time.Time, output chan<- *ct..ExpandedFormation) (stream.Stream, error)
}


type Config struct {
	Pin  []byte
	Domain string
}

type ErrNotFound = ct.ErrNotFound

func newClient(key string, url string, http *http.Client) *v1controller.Client {
	c := v1controller.Client{
		Client: &httpclient.Client{
			ErrNotFound: ErrNotFound,
			Key: 		key,
			URL:url,
			HTTP:http,
		},
	}
	return c
}