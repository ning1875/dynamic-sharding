package main

import (
	"gopkg.in/alecthomas/kingpin.v2"
	"github.com/prometheus/common/promlog"
	"github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"context"
	"github.com/oklog/run"
	"os"
	"os/signal"
	"syscall"
	"github.com/go-kit/kit/log/level"
	"dynamic-sharding/pkg/web"
	"github.com/gin-gonic/gin"
	"dynamic-sharding/pkg/config"
	"dynamic-sharding/pkg/sd"
	"fmt"
)

func main() {

	var (
		configFile = kingpin.Flag("config.file", "dynamic-sharding configuration file path.").Default("dynamic-sharding.yml").String()
	)

	// init logger
	promlogConfig := &promlog.Config{}
	flag.AddFlags(kingpin.CommandLine, promlogConfig)
	kingpin.Version(version.Print("dynamic-sharding"))
	kingpin.HelpFlag.Short('h')
	kingpin.Parse()
	logger := promlog.New(promlogConfig)

	// new grpc manager
	ctxAll, cancelAll := context.WithCancel(context.Background())
	sc, _ := config.LoadFile(*configFile, logger)

	// init consul client
	client, err := sd.NewConsulClient(sc.ConsulServer.Addr, logger)

	if err != nil || client == nil {
		level.Error(logger).Log("msg", "NewConsulClient Error, exiting ...", "error", err)
		return
	}

	// register service
	errors := sd.RegisterFromFile(client, sc.PGW.Servers, sc.ConsulServer.RegisterServiceName, sc.PGW.Port)
	if len(errors) > 0 {
		level.Error(logger).Log("msg", "RegisterFromFile Error", "error", errors)
	}
	// init node hash ring
	var ss []string
	for _, i := range sc.PGW.Servers {
		ss = append(ss, fmt.Sprintf("%s:%d", i, sc.PGW.Port))
	}

	sd.NewConsistentHashNodesRing(ss)
	var g run.Group
	{
		// Termination handler.
		term := make(chan os.Signal, 1)
		signal.Notify(term, os.Interrupt, syscall.SIGTERM)
		cancel := make(chan struct{})
		g.Add(

			func() error {
				select {
				case <-term:
					level.Warn(logger).Log("msg", "Received SIGTERM, exiting gracefully...")
					cancelAll()
					return nil
					//TODO clean work here
				case <-cancel:
					level.Warn(logger).Log("msg", "server finally exit...")
					return nil
				}
			},
			func(err error) {
				close(cancel)

			},
		)
	}

	{
		// metrics web handler.
		g.Add(func() error {
			level.Info(logger).Log("msg", "start web service Listening on address", "address", sc.HttpListenAddr)
			gin.SetMode(gin.ReleaseMode)
			routes := gin.Default()
			errchan := make(chan error)

			go func() {
				errchan <- web.StartGin(sc.HttpListenAddr, routes)
			}()
			select {
			case err := <-errchan:
				level.Error(logger).Log("msg", "Error starting HTTP server", "err", err)
				return err
			case <-ctxAll.Done():
				level.Info(logger).Log("msg", "Web service Exit..")
				return nil

			}

		}, func(err error) {
			cancelAll()
		})
	}

	{
		// service discovery   manager.
		g.Add(func() error {
			err := client.RunRefreshServiceNode(ctxAll, sc.ConsulServer.RegisterServiceName)
			if err != nil {
				level.Error(logger).Log("msg", "service discovery  error", "error", err)
			}
			return err
		}, func(err error) {
			cancelAll()
		})
	}
	g.Run()
}
