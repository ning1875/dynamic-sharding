package main

import (
	"fmt"
	"time"

	"context"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/oklog/run"
	"github.com/prometheus/common/promlog"
	promlogflag "github.com/prometheus/common/promlog/flag"
	"github.com/prometheus/common/version"
	"gopkg.in/alecthomas/kingpin.v2"

	"dynamic-sharding/pkg/config"
	"dynamic-sharding/pkg/sd"
	"dynamic-sharding/pkg/web"
)

func main() {

	var (
		app = kingpin.New(filepath.Base(os.Args[0]), "The dynamic-sharding")
		//configFile = kingpin.Flag("config.file", "docker-mon configuration file path.").Default("docker-mon.yml").String()
		configFile = app.Flag("config.file", "docker-mon configuration file path.").Default("dynamic-sharding.yml").String()
	)
	promlogConfig := promlog.Config{}

	app.Version(version.Print("dynamic-sharding"))
	app.HelpFlag.Short('h')
	promlogflag.AddFlags(app, &promlogConfig)
	kingpin.MustParse(app.Parse(os.Args[1:]))

	var logger log.Logger
	logger = func(config *promlog.Config) log.Logger {
		var (
			l  log.Logger
			le level.Option
		)
		if config.Format.String() == "logfmt" {
			l = log.NewLogfmtLogger(log.NewSyncWriter(os.Stderr))
		} else {
			l = log.NewJSONLogger(log.NewSyncWriter(os.Stderr))
		}

		switch config.Level.String() {
		case "debug":
			le = level.AllowDebug()
		case "info":
			le = level.AllowInfo()
		case "warn":
			le = level.AllowWarn()
		case "error":
			le = level.AllowError()
		}
		l = level.NewFilter(l, le)
		l = log.With(l, "ts", log.TimestampFormat(
			func() time.Time { return time.Now().Local() },
			"2006-01-02T15:04:05.000Z07:00",
		), "caller", log.DefaultCaller)
		return l
	}(&promlogConfig)

	// new grpc manager
	ctxAll, cancelAll := context.WithCancel(context.Background())
	sc, err := config.LoadFile(*configFile, logger)
	if err != nil {
		level.Error(logger).Log("msg", "config.LoadFil Error, exiting ...", "error", err)
		return
	}
	// init consul client
	client, err := sd.NewConsulClient(sc.ConsulServer.Addr, logger)

	if err != nil || client == nil {
		level.Error(logger).Log("msg", "NewConsulClient Error, exiting ...", "error", err)
		return
	}

	// init node hash ring
	var ss []string
	for _, i := range sc.PGW.Servers {
		ss = append(ss, fmt.Sprintf("%s:%d", i, sc.PGW.Port))
	}

	sd.NewConsistentHashNodesRing(ss)

	// register service
	errors := sd.RegisterFromFile(client, sc.PGW.Servers, sc.ConsulServer.RegisterServiceName, sc.PGW.Port)
	if len(errors) > 0 {
		level.Error(logger).Log("msg", "RegisterFromFile Error", "error", errors)
	}

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
			errchan := make(chan error, 1)

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
		// WatchService   manager.
		g.Add(func() error {
			err := client.RunRefreshServiceNode(ctxAll, sc.ConsulServer.RegisterServiceName, sc.ConsulServer.Addr)
			if err != nil {
				level.Error(logger).Log("msg", "watchService_error", "error", err)
			}
			return err
		}, func(err error) {
			cancelAll()
		})
	}
	g.Run()
}
