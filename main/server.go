package main

import (
	"runtime"
	"syscall"

	"github.com/Djoulzy/Polycom/hub"
	"github.com/Djoulzy/Polycom/nettools/httpserver"
	"github.com/Djoulzy/Polycom/nettools/scaling"
	"github.com/Djoulzy/Polycom/nettools/tcpserver"
	"github.com/Djoulzy/Polycom/urlcrypt"
	"github.com/Djoulzy/Polycom/world"

	"github.com/Djoulzy/Tools/clog"
	"github.com/Djoulzy/Tools/config"

	"github.com/Djoulzy/Polycom/monitoring"
)

var Cryptor *urlcrypt.Cypher

var HTTPManager httpserver.Manager
var TCPManager tcpserver.Manager
var ScaleList *scaling.ServersList

// var Storage *storage.Driver

var zeWorld *world.WORLD
var zeHub *hub.Hub

func maxOpenFiles() int {
	var rLimit syscall.Rlimit

	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		clog.Error("server", "maxOpenFiles", "Error Getting Rlimit %s", err)
	}

	if rLimit.Cur < rLimit.Max {
		rLimit.Cur = rLimit.Max
		err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		if err != nil {
			clog.Error("server", "maxOpenFiles", "Error Setting Rlimit %s", err)
		}
	}

	return int(rLimit.Cur)
}

func main() {
	config.Load("server.ini", conf)

	clog.LogLevel = conf.LogLevel
	clog.StartLogging = conf.StartLogging

	// System Optims
	clog.Output("Using %d CPUs.", runtime.GOMAXPROCS(runtime.NumCPU()))
	maxFiles := maxOpenFiles()
	clog.Output("Setting maxOpenFiles to %d.", maxFiles)
	////////////////

	totalConn := conf.MaxUsersConns + conf.MaxMonitorsConns + conf.MaxServersConns + conf.MaxIncommingConns
	if totalConn > maxFiles {
		conf.MaxUsersConns = maxFiles - 120
		conf.MaxMonitorsConns = 3
		conf.MaxServersConns = 10
		conf.MaxIncommingConns = 100
		clog.Warn("server", "main", "Setting MaxUser to %d.", conf.MaxUsersConns)
	}

	Cryptor = &urlcrypt.Cypher{
		HASH_SIZE: conf.HASH_SIZE,
		HEX_KEY:   []byte(conf.HEX_KEY),
		HEX_IV:    []byte(conf.HEX_IV),
	}

	zeHub = hub.NewHub()
	zeWorld = world.Init(zeHub)

	mon_params := &monitoring.Params{
		ServerID:          conf.Name,
		Httpaddr:          conf.HTTPaddr,
		Tcpaddr:           conf.TCPaddr,
		MaxUsersConns:     conf.MaxUsersConns,
		MaxMonitorsConns:  conf.MaxMonitorsConns,
		MaxServersConns:   conf.MaxServersConns,
		MaxIncommingConns: conf.MaxIncommingConns,
	}
	go monitoring.Start(zeHub, mon_params)

	tcp_params := &tcpserver.Manager{
		ServerName:               conf.Name,
		Tcpaddr:                  conf.TCPaddr,
		Hub:                      zeHub,
		ConnectTimeOut:           conf.ConnectTimeOut,
		WriteTimeOut:             conf.WriteTimeOut,
		ScalingCheckServerPeriod: conf.ScalingCheckServerPeriod,
		MaxServersConns:          conf.MaxServersConns,
		CallToAction:             CallToAction,
		Cryptor:                  Cryptor,
	}

	ScaleList = scaling.Init(tcp_params, &conf.KnownBrothers.Servers)
	go ScaleList.Start()
	// go scaling.Start(ScalingServers)

	http_params := &httpserver.Manager{
		ServerName:       conf.Name,
		Httpaddr:         conf.HTTPaddr,
		Hub:              zeHub,
		ReadBufferSize:   conf.ReadBufferSize,
		WriteBufferSize:  conf.WriteBufferSize,
		HandshakeTimeout: conf.HandshakeTimeout,
		NBAcceptBySecond: conf.NBAcceptBySecond,
		CallToAction:     CallToAction,
		Cryptor:          Cryptor,
		MapGenCallback:   zeWorld.GetMapArea,
		ClientDisconnect: zeWorld.DropUser,
	}
	clog.Output("HTTP Server starting listening on %s", conf.HTTPaddr)
	go HTTPManager.Start(http_params)

	clog.Output("TCP Server starting listening on %s", conf.TCPaddr)
	go TCPManager.Start(tcp_params)

	go zeWorld.Run()
	zeHub.Run()
}
