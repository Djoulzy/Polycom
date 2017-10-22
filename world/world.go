package world

import (
	"encoding/json"
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
	ID        string
	Type      string
	Face      string
	ComID     int
	X         int
	Y         int
	Speed     int
	waitState int
}

type User struct {
	MOB
}

type WORLD struct {
	hub     *hub.Hub
	MobList map[string]*MOB
}

func (W *WORLD) spawnMob() {
	if len(W.MobList) < 1 {
		uid, _ := uuid.NewV4()
		mob := &MOB{
			ID:        uid.String(),
			Type:      "M",
			X:         8 * 32,
			Y:         5 * 32,
			Speed:     50,
			waitState: 0,
		}
		W.MobList[mob.ID] = mob
		message := []byte(fmt.Sprintf("[NMOB]%s", mob.ID))
		clog.Info("WORLD", "spawnMob", "Spawning new mob %s", mob.ID)
		mess := hub.NewMessage(nil, hub.ClientUser, nil, message)
		W.hub.Broadcast <- mess
	}
}

func (W *WORLD) moveMob() {
	for _, mob := range W.MobList {
		if mob.waitState <= 0 {
			mob.X -= 32
			message := []byte(fmt.Sprintf("[BCST]{\"type\":\"%s\",\"id\":\"%s\",\"face\":\"z1\",\"num\":%d,\"move\":\"%s\",\"speed\":\"%d\",\"x\":%d,\"y\":%d}", mob.ID, mob.Type, 1, "left", mob.Speed, mob.X, mob.Y))
			mess := hub.NewMessage(nil, hub.ClientUser, nil, message)
			W.hub.Broadcast <- mess
			mob.waitState = mob.Speed
		} else {
			mob.waitState -= 1
		}
	}
}

func (W *WORLD) logUser(infos User) {
}

func (W *WORLD) CallToAction(message []byte) {
	var infos User
	err := json.Unmarshal(message, &infos)
	if err == nil {
		if infos.Type == "P" {
			W.logUser(infos)
		}
	} else {
		clog.Warn("World", "CallToAction", "%s", err)
	}
}

func (W *WORLD) Run() {
	ticker := time.NewTicker(timeStep)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case <-ticker.C:
			W.spawnMob()
			W.moveMob()
		}
	}
}

func Init(zeHub *hub.Hub) *WORLD {
	zeWorld := &WORLD{}
	zeWorld.MobList = make(map[string]*MOB)
	zeWorld.hub = zeHub

	return zeWorld
}
