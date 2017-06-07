package monitoring

import (
	"encoding/json"
	"fmt"
	"math"
	"runtime"
	"time"

	clog "github.com/Djoulzy/Polycom/CLog"

	"github.com/Djoulzy/Polycom/Hub"
	"github.com/shirou/gopsutil/cpu"
	"github.com/shirou/gopsutil/load"
	"github.com/shirou/gopsutil/mem"
)

const statsTimer = 5 * time.Second

type ClientList map[string]*Hub.Client

type Brother struct {
	Tcpaddr  string
	Httpaddr string
}

type ServerMetrics struct {
	SID      string
	TCPADDR  string
	HTTPADDR string
	HOST     string
	CPU      int
	GORTNE   int
	STTME    string
	UPTME    string
	LSTUPDT  string
	LAVG     int
	MEM      string
	SWAP     string
	NBMESS   int
	NBI      int
	MXI      int
	NBU      int
	MXU      int
	NBM      int
	MXM      int
	NBS      int
	MXS      int
	BRTHLST  map[string]Brother
}

type BrotherList struct {
	BRTHLST map[string]Brother
}

type ClientsRegister struct {
	ID   ClientList
	Name ClientList
	Type map[int]ClientList
}

type Params struct {
	ServerID          string
	Httpaddr          string
	Tcpaddr           string
	MaxUsersConns     int
	MaxMonitorsConns  int
	MaxServersConns   int
	MaxIncommingConns int
}

var StartTime time.Time
var UpTime time.Duration
var MachineLoad *load.AvgStat
var nbcpu int
var cr ClientsRegister
var AddBrother = make(chan map[string]Brother)
var brotherlist = make(map[string]Brother)

func getMemUsage() string {
	v, _ := mem.VirtualMemory()
	return fmt.Sprintf("<th>Mem</th><td class='memCell'>%v Mo</td><td class='memCell'>%v Mo</td><td class='memCell'>%.1f%%</td>", (v.Total / 1048576), (v.Free / 1048576), v.UsedPercent)
}

func getSwapUsage() string {
	v, _ := mem.SwapMemory()
	return fmt.Sprintf("<th>Swap</th><td class='memCell'>%v Mo</td><td class='memCell'>%v Mo</td><td class='memCell'>%.1f%%</td>", (v.Total / 1048576), (v.Free / 1048576), v.UsedPercent)
}

func addToBrothersList(srv map[string]Brother) {
	for name, infos := range srv {
		brotherlist[name] = infos
	}
}

func LoadAverage(hub *Hub.Hub, p *Params) {
	ticker := time.NewTicker(statsTimer)
	MachineLoad = &load.AvgStat{0, 0, 0}
	nbcpu, _ := cpu.Counts(true)
	StartTime = time.Now()

	for {
		select {
		case newSrv := <-AddBrother:
			addToBrothersList(newSrv)
		case <-ticker.C:
			tmp, _ := load.Avg()
			MachineLoad = tmp
			loadIndice := int(math.Ceil((((MachineLoad.Load1*5 + MachineLoad.Load5*3 + MachineLoad.Load15*2) / 10) / float64(nbcpu)) * 100))
			// mess := NewMessage(nil, machineLoad.String())
			t := time.Now()
			UpTime = time.Since(StartTime)

			newStats := ServerMetrics{
				SID:      p.ServerID,
				TCPADDR:  p.Tcpaddr,
				HTTPADDR: p.Httpaddr,
				HOST:     fmt.Sprintf("HTTP: %s - TCP: %s", p.Httpaddr, p.Tcpaddr),
				CPU:      nbcpu,
				GORTNE:   runtime.NumGoroutine(),
				STTME:    StartTime.Format("02/01/2006 15:04:05"),
				UPTME:    UpTime.String(),
				LSTUPDT:  t.Format("02/01/2006 15:04:05"),
				LAVG:     loadIndice,
				MEM:      getMemUsage(),
				SWAP:     getSwapUsage(),
				NBMESS:   hub.SentMessByTicks,
				NBI:      len(hub.Incomming),
				MXI:      p.MaxIncommingConns,
				NBU:      len(hub.Users),
				MXU:      p.MaxUsersConns,
				NBM:      len(hub.Monitors),
				MXM:      p.MaxMonitorsConns,
				NBS:      len(hub.Servers),
				MXS:      p.MaxServersConns,
				BRTHLST:  brotherlist,
			}

			newBrthList := BrotherList{
				BRTHLST: brotherlist,
			}

			// clog.Test("monitoring", "addToBrothersList", "Brother List: %s", brotherlist)
			brth_json, _ := json.Marshal(newBrthList)
			json, err := json.Marshal(newStats)
			if err != nil {
				clog.Error("Monitoring", "LoadAverage", "MON: Cannot send server metrics to listeners ...")
			} else {
				if len(hub.Monitors)+len(hub.Servers) > 0 {
					hub.SentMessByTicks = 0
					// clog.Trace("Monitoring", "LoadAverage", "%s", json)
					mess := Hub.NewMessage(Hub.ClientMonitor, nil, json)
					hub.Status <- mess
					mess = Hub.NewMessage(Hub.ClientServer, nil, append([]byte("MON|"), json...))
					hub.Status <- mess
					mess = Hub.NewMessage(Hub.ClientUser, nil, append([]byte("FLLBCKSRV|"), brth_json...))
					hub.Broadcast <- mess
				}
			}
		}
	}
	defer func() {
		ticker.Stop()
	}()
}

func Start(hub *Hub.Hub, p *Params) {
	// addToBrothersList(list)
	LoadAverage(hub, p)
}
