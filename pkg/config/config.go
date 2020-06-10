package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"github.com/go-kit/kit/log/level"
	"github.com/go-kit/kit/log"
)

type Config struct {
	ConsulServer   *ConsulServerConfig `yaml:"consul_server"`
	HttpListenAddr string              `yaml:"http_listen_addr"`
	PGW            *PushGateWayConfig  `yaml:"pushgateway"`
}

type ConsulServerConfig struct {
	Addr                string `yaml:"addr,omitempty"`
	Username            string `yaml:"username,omitempty"`
	Password            string `yaml:"password,omitempty"`
	RegisterServiceName string `yaml:"register_service_name,omitempty"`
}

type PushGateWayConfig struct {
	Servers []string `yaml:"servers"`
	Port    int      `yaml:"port"`
}

func Load(s string) (*Config, error) {
	cfg := &Config{}

	err := yaml.UnmarshalStrict([]byte(s), cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func LoadFile(filename string, logger log.Logger) (*Config, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	cfg, err := Load(string(content))
	if err != nil {
		level.Error(logger).Log("msg", "parsing YAML file errr...", "error", err)
	}
	return cfg, nil
}
