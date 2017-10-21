package world

import (
	"fmt"
	"time"

	"github.com/Djoulzy/Polycom/hub"
	"github.com/Djoulzy/Tools/clog"
	"github.com/nu7hatch/gouuid"
)

const (
	timeStep = 100 * time.Millisecond // Actualisation 10 par seconde
)

type MOB struct {
	MobID     string
	MobType   int
	x         int
	y         int
	speed     int
	waitState int
}

type WORLD struct {
	hub     *hub.Hub
	MobList map[string]*MOB
}

func (W *WORLD) spawnMob() {
	if len(W.MobList) < 1 {
		uid, _ := uuid.NewV4()
		mob := &MOB{
			MobID:     uid.String(),
			MobType:   1,
			x:         8 * 32,
			y:         5 * 32,
			speed:     100,
			waitState: 0,
		}
		W.MobList[mob.MobID] = mob
		message := []byte(fmt.Sprintf("[NMOB]%s", mob.MobID))
		clog.Info("WORLD", "spawnMob", "Spawning new mob %s", mob.MobID)
		mess := hub.NewMessage(nil, hub.ClientUser, nil, message)
		W.hub.Broadcast <- mess
	}
}

func (W *WORLD) moveMob() {
	for _, mob := range W.MobList {
		if mob.waitState <= 0 {
			mob.x -= 32
			message := []byte(fmt.Sprintf("[BCST]{\"type\":\"M\",\"id\":\"%s\",\"face\":\"z1\",\"num\":%d,\"move\":\"%s\",\"x\":%d,\"y\":%d}", mob.MobID, 1, "left", mob.x, mob.y))
			mess := hub.NewMessage(nil, hub.ClientUser, nil, message)
			W.hub.Broadcast <- mess
			mob.waitState = mob.speed
		} else {
			mob.waitState -= 1
		}
	}
}

func (W *WORLD) Init(zeHub *hub.Hub) {
	W.MobList = make(map[string]*MOB)
	W.hub = zeHub
}

func Start(zeHub *hub.Hub) {
	ticker := time.NewTicker(timeStep)
	defer func() {
		ticker.Stop()
	}()

	zeWorld := &WORLD{}
	zeWorld.Init(zeHub)

	for {
		select {
		case <-ticker.C:
			zeWorld.spawnMob()
			zeWorld.moveMob()
		}
	}
}
