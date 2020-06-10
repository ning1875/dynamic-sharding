package sd

import (
	"fmt"
	consul "github.com/hashicorp/consul/api"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	time "time"
	"context"
)

const GetServiceInterval = time.Second * 5

type client struct {
	consul *consul.Client
	logger log.Logger
}

type Client interface {
	// Get a Service from consul
	GetService(string, string) ([]string, error)
	// register a service with local agent
	ServiceRegister(string, string, int) error
	// Deregister a service with local agent
	DeRegister(string) error
}

func NewConsulClient(addr string, logger log.Logger) (*client, error) {
	config := consul.DefaultConfig()
	config.Address = addr
	c, err := consul.NewClient(config)
	if err != nil {
		return nil, err
	}
	return &client{consul: c, logger: logger}, nil
}

// Register a service with consul local agent
func (c *client) ServiceRegister(srv_name, srv_host string, srv_port int) error {

	reg := new(consul.AgentServiceRegistration)
	reg.Name = srv_name

	thisId := fmt.Sprintf("%s_%d", srv_host, srv_port)
	reg.ID = thisId
	reg.Port = srv_port
	//registration.Tags = []string{"user-tomcat"}
	reg.Address = srv_host
	level.Info(c.logger).Log("msg", "ServiceRegisterStart", "id", thisId)
	//增加check
	check := new(consul.AgentServiceCheck)
	check.HTTP = fmt.Sprintf("http://%s:%d%s", reg.Address, reg.Port, "/-/healthy")
	//设置超时 5s。
	check.Timeout = "2s"
	check.DeregisterCriticalServiceAfter = "5s"
	//设置间隔 5s。
	check.Interval = "5s"
	//注册check服务。
	reg.Check = check

	return c.consul.Agent().ServiceRegister(reg)
}

// DeRegister a service with consul local agent
func (c *client) DeRegister(id string) error {
	return c.consul.Agent().ServiceDeregister(id)
}

// Service return a service
func (c *client) GetService(service, tag string) ([]string, error) {
	passingOnly := true
	addrs, _, err := c.consul.Health().Service(service, tag, passingOnly, nil)
	if len(addrs) == 0 && err == nil {
		return nil, fmt.Errorf("service ( %s ) was not found", service)
	}

	if err != nil {
		return nil, err
	}
	var hs []string

	for _, a := range addrs {

		hs = append(hs, fmt.Sprintf("%s:%d", a.Service.Address, a.Service.Port))
	}
	if len(hs) > 0 {
		NodeUpdateChan <- hs
	}

	return hs, nil
}

func RegisterFromFile(c *client, servers []string, srv_name string, srv_port int) (errors []error) {

	for _, addr := range servers {

		e := c.ServiceRegister(srv_name, addr, srv_port)
		if e != nil {
			errors = append(errors, e)
		}

	}
	return
}
func (c *client) RunRefreshServiceNode(ctx context.Context, srv_name string) error {
	ticker := time.NewTicker(GetServiceInterval)
	level.Info(c.logger).Log("msg", "RunRefreshServiceNode start....")
	go RunReshardHashRing(ctx, c.logger)
	c.GetService(srv_name, "")
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			level.Info(c.logger).Log("msg", "receive_quit_signal_and_quit")
			return nil
		case <-ticker.C:
			c.GetService(srv_name, "")
		}

	}
	return nil
}
