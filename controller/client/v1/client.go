package v1controller

import (
	"weo/pkg/httpclient"
	"weo/pkg/stream"
)

type Client struct {
	*httpclient.Client
}

func (c *Client) SetKey(newKey string) {
	c.Key = newKey
}

type jobWatcher struct {
	events 		chan *ct.Job
	stream 		stream.Stream
	releaseID 	string
}