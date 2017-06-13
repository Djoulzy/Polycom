package main

import (
	clog "github.com/Djoulzy/Polycom/CLog"
	urlcrypt "github.com/Djoulzy/Polycom/URLCrypt"
	scaling "github.com/Djoulzy/Polycom/nettools/Scaling"
	"github.com/Djoulzy/Polycom/storage"

	"github.com/Djoulzy/Polycom/Config"
	"github.com/Djoulzy/Polycom/Hub"
	"github.com/Djoulzy/Polycom/monitoring"
	"github.com/Djoulzy/Polycom/nettools/HTTPServer"
	"github.com/Djoulzy/Polycom/nettools/TCPServer"
)

var conf *Config.Data

var cryptor *urlcrypt.Cypher

var HTTPManager HTTPServer.Manager
var TCPManager TCPServer.Manager
var scaleList *scaling.ServersList
var Storage *storage.Driver

func main() {
	conf, _ = Config.Load()

	clog.LogLevel = conf.LogLevel
	clog.StartLogging = conf.StartLogging

	cryptor = &urlcrypt.Cypher{
		HASH_SIZE: conf.HASH_SIZE,
		HEX_KEY:   []byte(conf.HEX_KEY),
		HEX_IV:    []byte(conf.HEX_IV),
	}

	hub := Hub.NewHub()
	go hub.Run()

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
	go monitoring.Start(hub, mon_params)

	tcp_params := &TCPServer.Manager{
		ServerName:               conf.Name,
		Tcpaddr:                  conf.TCPaddr,
		Hub:                      hub,
		ConnectTimeOut:           conf.ConnectTimeOut,
		WriteTimeOut:             conf.WriteTimeOut,
		ScalingCheckServerPeriod: conf.ScalingCheckServerPeriod,
		MaxServersConns:          conf.MaxServersConns,
	}

	scaleList = scaling.Init(tcp_params, &conf.KnownBrothers.Servers)
	go scaleList.Start()
	// go scaling.Start(ScalingServers)

	http_params := &HTTPServer.Manager{
		ServerName:       conf.Name,
		Httpaddr:         conf.HTTPaddr,
		Hub:              hub,
		ReadBufferSize:   conf.ReadBufferSize,
		WriteBufferSize:  conf.WriteBufferSize,
		HandshakeTimeout: conf.HandshakeTimeout,
	}
	clog.Output("HTTP Server starting listening on %s", conf.HTTPaddr)
	go HTTPManager.Start(http_params, CallToActionHTTP)

	clog.Output("TCP Server starting listening on %s", conf.TCPaddr)
	TCPManager.Start(tcp_params, CallToActionTCP)
}
