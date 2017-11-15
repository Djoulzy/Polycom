package world

import (
	"time"

	"github.com/Djoulzy/Polycom/hub"
	"github.com/Djoulzy/Polycom/world/pathfinder"
)

const (
	timeStep  = 100 * time.Millisecond // Actualisation 10 par seconde
	tileSize  = 32
	AOIWidth  = 10
	AOIHeight = 10
	mobSpeed  = 8
	maxMobNum = 10
)

type Entity struct {
	ID        string `bson:"id" json:"id"`
	Type      string `bson:"typ" json:"typ"`
	Face      string `bson:"png" json:"png"`
	ComID     int    `bson:"num" json:"num"`
	Dir       string `bson:"mov" json:"mov"`
	X         int    `bson:"x" json:"x"` // Col nums
	Y         int    `bson:"y" json:"y"` // Row nums
	Pow       int    `bson:"pow" json:"pow"`
	Speed     int    `bson:"spd" json:"spd"`
	waitState int
}

type Attributes struct {
	PV     int `bson:"pv" json:"pv"`
	Starv  int `bson:"st" json:"st"`
	Thirst int `bson:"th" json:"th"`
	Fight  int `bson:"fgt" json:"fgt"`
	Shoot  int `bson:"sht" json:"sht"`
	Craft  int `bson:"cft" json:"cft"`
	Breed  int `bson:"brd" json:"brd"`
	Grow   int `bson:"grw" json:"grw"`
}

type USER struct {
	HubClient *hub.Client
	Entity
	Attributes
}

type MOB struct {
	Entity
}

type TILE struct {
	Type int
	ID   string
}

type WORLD struct {
	hub       *hub.Hub
	MobList   map[string]*MOB
	UserList  map[string]*USER
	Width     int
	Height    int
	Map       pathfinder.MapData
	EntityMap [][]interface{}
	Graph     *pathfinder.Graph
	AOIs      *AOIList
}

type FILELAYER struct {
	Data   []int  `bson:"data" json:"data"`
	Name   string `bson:"name" json:"name"`
	Width  int    `bson:"width" json:"width"`
	Height int    `bson:"height" json:"height"`
}

type FILEMAP struct {
	Layers []FILELAYER `bson:"layers" json:"layers"`
}
