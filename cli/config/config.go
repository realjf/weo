package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/mitchellh/go-homedir"
	"os"
	"path/filepath"
	"runtime"
	"weo/controller/client"
)

var ErrNoDockerPushURL = errors.New("ERROR: Docker push URL not configured, set it with 'weo docker set-push-url'")

type Cluster struct {
	Name		string `json:"name"`
	Key			string `json:"key"`
	TLSPin		string `json:"tls_pin" toml:"TLSPin,omitempty"`
	ControllerURL string `json:"controller_url"`
	GitURL        string `json:"git_url"`
	ImageURL      string `json:"image_url"`
	DockerPushURL string `json:"docker_push_url"`
}


func (c *Cluster) Client() (controller.Client, error) {
	var pin []byte
	if c.TLSPin != "" {
		var err error
		pin, err = base64.StdEncoding.DecodeString(c.TLSPin)
		if err != nil {
			return nil, fmt.Errorf("error decoding tls pin: %s", err)
		}
	}

	return controller.NewClientWithConfig(c.ControllerURL, c.Key, controller.Config{Pin: pin})
}

func (c *Cluster) Client() (controller.Client, error) {
	var pin []byte
	if c.TLSPin != "" {
		var err error
		pin, err = base64.StdEncoding.DecodeString(c.TLSPin)
		if err != nil {
			return nil, fmt.Errorf("error decoding tls pin: %s", err)
		}
	}
	return controller.NewClientWithConfig(c.ControllerURL, c.Key, controller.Config{Pin: pin})
}

func (c *Cluster) TarClient() (*tarclient.Client, error) {
	if c.ImageURL == "" {
		return nil, errors.New("cluster: missing ImageURL .weorc config")
	}
	var pin []byte
	if c.TLSPin != "" {
		var err error
		pin, err = base64.StdEncoding.DecodeString(c.TLSPin)
		if err != nil {
			return nil, fmt.Errorf("error decoding tls pin: %s", err)
		}
	}
	return tarclient.NewClientWithConfig(c.ImageURL, c.Key, tarclient.Config{Pin: pin}), nil
}


type Config struct {
	Default  string     `toml:"default"`
	Clusters []*Cluster `toml:"cluster"`
}

func HomeDir() string {
	dir, err := homedir.Dir()
	if err != nil {
		panic(err)
	}
	return dir
}


func Dir() string {
	if runtime.GOOS == "windows" {
		return filepath.Join(os.Getenv("APPDATA"), "weo")
	}
	return filepath.Join(HomeDir(), ".weo")
}



