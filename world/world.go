package world

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
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
	Dir       string
	X         int
	Y         int
	Speed     int
	waitState int
}

type User struct {
	MOB
}

type WORLD struct {
	hub      *hub.Hub
	MobList  map[string]*MOB
	UserList map[string]*User
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

func (W *WORLD) findCloserUser(mob *MOB) (*User, error) {
	var distFound float64 = 0
	var userFound *User = nil
	for _, player := range W.UserList {
		largeur := math.Abs(float64(mob.X - player.X))
		hauteur := math.Abs(float64(mob.Y - player.Y))
		dist := math.Sqrt(math.Pow(largeur, 2) + math.Pow(hauteur, 2))
		if dist < distFound || distFound == 0 {
			userFound = player
			distFound = dist
		}
	}
	if userFound != nil {
		return userFound, nil
	} else {
		return nil, errors.New("No prey")
	}
}

func (W *WORLD) moveMob(mob *MOB) {
	prey, err := W.findCloserUser(mob)
	if err == nil {
		clog.Info("World", "moveMob", "Seeking for %s", prey.ID)
		if math.Abs(float64(prey.X-mob.X)) < math.Abs(float64(prey.Y-mob.Y)) {
			if mob.Y > prey.Y {
				mob.Y -= 32
				mob.Dir = "up"
			} else {
				mob.Y += 32
				mob.Dir = "down"
			}
		} else {
			if mob.X > prey.X {
				mob.X -= 32
				mob.Dir = "left"
			} else {
				mob.X += 32
				mob.Dir = "right"
			}
		}
		message := []byte(fmt.Sprintf("[BCST]{\"type\":\"%s\",\"id\":\"%s\",\"face\":\"z1\",\"num\":%d,\"move\":\"%s\",\"speed\":\"%d\",\"x\":%d,\"y\":%d}", mob.ID, mob.Type, 1, mob.Dir, mob.Speed, mob.X, mob.Y))
		mess := hub.NewMessage(nil, hub.ClientUser, nil, message)
		W.hub.Broadcast <- mess
		mob.waitState = mob.Speed
	}
}

func (W *WORLD) browseMob() {
	for _, mob := range W.MobList {
		if mob.waitState <= 0 {
			W.moveMob(mob)
		} else {
			mob.waitState -= 1
		}
	}
}

func (W *WORLD) logUser(infos *User) {
	if W.UserList[infos.ID] == nil {
		W.UserList[infos.ID] = infos
	}
}

func (W *WORLD) CallToAction(message []byte) {
	var infos User
	err := json.Unmarshal(message, &infos)
	if err == nil {
		if (infos.Type == "P") && (W.UserList[infos.ID]) == nil {
			clog.Warn("World", "CallToAction", "Registering user %s", infos.ID)
			W.UserList[infos.ID] = &infos
		} else {
			W.UserList[infos.ID].X = infos.X
			W.UserList[infos.ID].Y = infos.Y
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
			W.browseMob()
		}
	}
}

func Init(zeHub *hub.Hub) *WORLD {
	zeWorld := &WORLD{}
	zeWorld.MobList = make(map[string]*MOB)
	zeWorld.UserList = make(map[string]*User)
	zeWorld.hub = zeHub

	return zeWorld
}
