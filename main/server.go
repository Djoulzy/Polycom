package main

import (
	"github.com/Djoulzy/Polycom/clog"
	"github.com/Djoulzy/Polycom/hub"
	"github.com/Djoulzy/Polycom/nettools/httpserver"
	"github.com/Djoulzy/Polycom/nettools/scaling"
	"github.com/Djoulzy/Polycom/nettools/tcpserver"
	"github.com/Djoulzy/Polycom/storage"
	"github.com/Djoulzy/Polycom/urlcrypt"

	"github.com/Djoulzy/Polycom/config"
	"github.com/Djoulzy/Polycom/monitoring"
)

var conf *Config.Data

var Cryptor *urlcrypt.Cypher

var HTTPManager httpserver.Manager
var TCPManager tcpserver.Manager
var ScaleList *scaling.ServersList
var Storage *storage.Driver

func main() {
	conf, _ = Config.Load()

	clog.LogLevel = conf.LogLevel
	clog.StartLogging = conf.StartLogging

	Cryptor = &urlcrypt.Cypher{
		HASH_SIZE: conf.HASH_SIZE,
		HEX_KEY:   []byte(conf.HEX_KEY),
		HEX_IV:    []byte(conf.HEX_IV),
	}

	h := hub.NewHub()
	go h.Run()

	Storage = storage.Init()

	mon_params := &monitoring.Params{
		ServerID:          conf.Name,
		Httpaddr:          conf.HTTPaddr,
		Tcpaddr:           conf.TCPaddr,
		MaxUsersConns:     conf.MaxUsersConns,
		MaxMonitorsConns:  conf.MaxMonitorsConns,
		MaxServersConns:   conf.MaxServersConns,
		MaxIncommingConns: conf.MaxIncommingConns,
	}
	go monitoring.Start(h, mon_params)

	tcp_params := &tcpserver.Manager{
		ServerName:               conf.Name,
		Tcpaddr:                  conf.TCPaddr,
		Hub:                      h,
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
		Hub:              h,
		ReadBufferSize:   conf.ReadBufferSize,
		WriteBufferSize:  conf.WriteBufferSize,
		HandshakeTimeout: conf.HandshakeTimeout,
		CallToAction:     CallToAction,
		Cryptor:          Cryptor,
	}
	clog.Output("HTTP Server starting listening on %s", conf.HTTPaddr)
	go HTTPManager.Start(http_params)

	clog.Output("TCP Server starting listening on %s", conf.TCPaddr)
	TCPManager.Start(tcp_params)
}
