package sd

import (
	"fmt"
	"context"
	"strings"

	consul "github.com/hashicorp/consul/api"
	"github.com/hashicorp/consul/api/watch"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
)

type client struct {
	consul *consul.Client
	logger log.Logger
}

type Client interface {
	// Get a Service from consul
	//GetService(string, string) ([]string, error)
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
func (c *client) ServiceRegister(srvName, srvHost string, srvPort int) error {

	reg := new(consul.AgentServiceRegistration)
	reg.Name = srvName

	thisId := fmt.Sprintf("%s_%d", srvHost, srvPort)
	reg.ID = thisId
	reg.Port = srvPort
	reg.Address = srvHost
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

//// Service return a service
//func (c *client) GetService(service, tag string) ([]string, error) {
//	passingOnly := true
//	addrs, _, err := c.consul.Health().Service(service, tag, passingOnly, nil)
//	if len(addrs) == 0 && err == nil {
//		return nil, fmt.Errorf("service ( %s ) was not found", service)
//	}
//
//	if err != nil {
//		return nil, err
//	}
//	var hs []string
//
//	for _, a := range addrs {
//
//		hs = append(hs, fmt.Sprintf("%s:%d", a.Service.Address, a.Service.Port))
//	}
//	if len(hs) > 0 {
//		NodeUpdateChan <- hs
//	}
//
//	return hs, nil
//}

func RegisterFromFile(c *client, servers []string, srvName string, srvPort int) (errors []error) {

	for _, addr := range servers {

		e := c.ServiceRegister(srvName, addr, srvPort)
		if e != nil {
			errors = append(errors, e)
		}

	}
	return
}
func (c *client) RunRefreshServiceNode(ctx context.Context, srvName string, consulServerAddr string) error {
	level.Info(c.logger).Log("msg", "RunRefreshServiceNode start....")
	go RunReshardHashRing(ctx, c.logger)

	errchan := make(chan error, 1)
	go func() {
		errchan <- c.WatchService(ctx, srvName, consulServerAddr)

	}()
	select {
	case <-ctx.Done():
		level.Info(c.logger).Log("msg", "RunRefreshServiceNode_receive_quit_signal_and_quit")
		return nil
	case err := <-errchan:
		level.Error(c.logger).Log("msg", "WatchService_get_error", "err", err)
		return err
	}
	return nil
}

func (c *client) WatchService(ctx context.Context, srvName string, consulServerAddr string) error {

	watchConfig := make(map[string]interface{})

	watchConfig["type"] = "service"
	watchConfig["service"] = srvName
	watchConfig["handler_type"] = "script"
	watchConfig["passingonly"] = true
	watchPlan, err := watch.Parse(watchConfig)
	if err != nil {
		level.Error(c.logger).Log("msg", "create_Watch_by_watch_config_error", "srv_name", srvName, "error", err)
		return err

	}

	watchPlan.Handler = func(lastIndex uint64, result interface{}) {
		if entries, ok := result.([]*consul.ServiceEntry); ok {
			var hs []string

			for _, a := range entries {

				hs = append(hs, fmt.Sprintf("%s:%d", a.Service.Address, a.Service.Port))
			}
			if len(hs) > 0 {
				level.Info(c.logger).Log("msg", "service_node_change_by_healthy_check", "srv_name", srvName, "num", len(hs), "detail", strings.Join(hs, " "))
				NodeUpdateChan <- hs
			}

		}

	}
	if err := watchPlan.Run(consulServerAddr); err != nil {
		level.Error(c.logger).Log("msg", "watchPlan_run_error", "srv_name", srvName, "error", err)
		return err
	}
	return nil

}
