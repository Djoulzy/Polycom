package main

type ServerID struct {
	Name string
}

type Globals struct {
	LogLevel     int
	StartLogging bool
}

type ConnectionLimit struct {
	MaxUsersConns     int
	MaxMonitorsConns  int
	MaxServersConns   int
	MaxIncommingConns int
}

type ServersAddresses struct {
	HTTPaddr string
	TCPaddr  string
}

type KnownBrothers struct {
	Servers map[string]string
}

type HTTPServerConfig struct {
	ReadBufferSize   int
	WriteBufferSize  int
	HandshakeTimeout int
}

type TCPServerConfig struct {
	ConnectTimeOut           int
	WriteTimeOut             int
	ScalingCheckServerPeriod int
}

type Encryption struct {
	HASH_SIZE int
	HEX_KEY   string
	HEX_IV    string
}

type AppConfig struct {
	ServerID
	Globals
	ConnectionLimit
	ServersAddresses
	KnownBrothers
	HTTPServerConfig
	TCPServerConfig
	Encryption
}

var conf *AppConfig = &AppConfig{
	ServerID{},
	Globals{
		LogLevel:     4,
		StartLogging: true,
	},
	ConnectionLimit{
		MaxUsersConns:     100,
		MaxMonitorsConns:  3,
		MaxServersConns:   5,
		MaxIncommingConns: 50,
	},
	ServersAddresses{
		HTTPaddr: "localhost:8080",
		TCPaddr:  "localhost:8081",
	},
	KnownBrothers{},
	HTTPServerConfig{
		ReadBufferSize:   4096,
		WriteBufferSize:  4096,
		HandshakeTimeout: 5,
	},
	TCPServerConfig{
		ConnectTimeOut:           2,
		WriteTimeOut:             1,
		ScalingCheckServerPeriod: 10,
	},
	Encryption{
		HASH_SIZE: 8,
		HEX_KEY:   "0000000000000000000000000000000000000000000000000000000000000000",
		HEX_IV:    "00000000000000000000000000000000",
	},
}
