package world

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/Djoulzy/Polycom/hub"
	"github.com/Djoulzy/Polycom/storage"
	"github.com/Djoulzy/Tools/clog"
	"github.com/nu7hatch/gouuid"
)

func (W *WORLD) findSpawnPlace() (int, int) {
	for {
		x := rand.Intn(W.Map.Width)
		y := rand.Intn(W.Map.Height)
		if W.tileIsFree(x, y) {
			return x, y
		}
	}
}

func (W *WORLD) spawnMob() {
	if len(W.MobList) < maxMobNum {
		rand.Seed(time.Now().UnixNano())
		face := fmt.Sprintf("%d", rand.Intn(8))
		uid, _ := uuid.NewV4()
		mob := &MOB{
			Entity{
				ID:        uid.String(),
				Type:      "M",
				Face:      face,
				ComID:     1,
				Speed:     mobSpeed,
				waitState: 0,
			},
		}
		mob.X, mob.Y = W.findSpawnPlace()
		W.Map.Entities[mob.X][mob.Y] = mob
		W.MobList[mob.ID] = mob
		message := []byte(fmt.Sprintf("[NMOB]%s", mob.ID))
		W.AOIs.addEvent(mob.X, mob.Y, message)
		// clog.Info("WORLD", "spawnMob", "Spawning new mob %s", mob.ID)
		// mess := hub.NewMessage(nil, hub.ClientUser, nil, message)
		// W.hub.Broadcast <- mess
	}
}

func (W *WORLD) findCloserUser(mob *MOB) (*USER, error) {
	var distFound float64 = 0
	var userFound *USER = nil
	for _, player := range W.UserList {
		largeur := math.Abs(float64(mob.X - player.X))
		hauteur := math.Abs(float64(mob.Y - player.Y))
		dist := math.Sqrt(math.Pow(largeur, 2) + math.Pow(hauteur, 2))
		if dist > 20 {
			continue
		}
		if dist == 0 {
			return nil, errors.New("Prey Catch")
		}
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

func (W *WORLD) sendMobPos(mob *MOB) {
	json, _ := json.Marshal(mob)
	message := []byte(fmt.Sprintf("[BCST]%s", json))
	W.AOIs.addEvent(mob.X, mob.Y, message)
	mob.waitState = mob.Speed
}

func (W *WORLD) tileIsFree(x, y int) bool {
	if W.Map.Block[x][y] == 0 && W.Map.Entities[x][y] == nil {
		return true
	}
	return false
}

func (W *WORLD) moveSIMPLE(mob *MOB, prey *USER) {
	// clog.Info("World", "moveMob", "Seeking for %s", prey.ID)
	// if math.Abs(float64(prey.X-mob.X)) < math.Abs(float64(prey.Y-mob.Y)) {
	if mob.Y > prey.Y && W.tileIsFree(mob.X, mob.Y-1) {
		W.Map.Entities[mob.X][mob.Y] = nil
		mob.Y -= 1
		mob.Dir = "up"
		W.sendMobPos(mob)
		W.Map.Entities[mob.X][mob.Y] = mob
		return
	}
	if mob.Y < prey.Y && W.tileIsFree(mob.X, mob.Y+1) {
		W.Map.Entities[mob.X][mob.Y] = nil
		mob.Y += 1
		mob.Dir = "down"
		W.sendMobPos(mob)
		W.Map.Entities[mob.X][mob.Y] = mob
		return
	}
	if mob.X > prey.X && W.tileIsFree(mob.X-1, mob.Y) {
		W.Map.Entities[mob.X][mob.Y] = nil
		mob.X -= 1
		mob.Dir = "left"
		W.sendMobPos(mob)
		W.Map.Entities[mob.X][mob.Y] = mob
		return
	}
	if mob.X < prey.X && W.tileIsFree(mob.X+1, mob.Y) {
		W.Map.Entities[mob.X][mob.Y] = nil
		mob.X += 1
		mob.Dir = "right"
		W.sendMobPos(mob)
		W.Map.Entities[mob.X][mob.Y] = mob
		return
	}
}

// func (W *WORLD) moveASTAR(mob *MOB, prey *USER) {
// 	node := W.getShortPath(mob, prey)
// 	if node != nil {
// 		clog.Info("World", "moveMob", "Seeking for %s", prey.ID)
// 		if node.X > mob.X {
// 			mob.Dir = "right"
// 		} else if node.X < mob.X {
// 			mob.Dir = "left"
// 		} else if node.Y < mob.Y {
// 			mob.Dir = "up"
// 		} else if node.Y > mob.Y {
// 			mob.Dir = "down"
// 		}
// 		mob.X = node.X
// 		mob.Y = node.Y
// 		W.sendMobPos(mob)
// 	}
// }

func (W *WORLD) moveMob(mob *MOB) {
	prey, err := W.findCloserUser(mob)
	if err == nil {
		// W.moveASTAR(mob, prey)
		W.moveSIMPLE(mob, prey)
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

func (W *WORLD) DropUser(id string) {
	user := W.UserList[id]
	dat, _ := json.Marshal(user)
	storage.SaveUser(id, dat)
	message := []byte(fmt.Sprintf("[KILL]%s", id))
	W.AOIs.addEvent(user.X, user.Y, message)

	W.Map.Entities[user.X][user.Y] = nil
	delete(W.UserList, id)
}

func (W *WORLD) LogUser(c *hub.Client) ([]byte, error) {
	var infos *USER
	dat, err := storage.LoadUser(c.Name)
	if err != nil {
		infos = &USER{
			Entity: Entity{
				ID: c.Name, Type: "P", Face: "h1", Dir: "down", X: 25, Y: 25,
			},
			Attributes: Attributes{
				PV: 15, Starv: 15, Thirst: 15,
			},
		}
		dat, err = json.Marshal(infos)
		if err != nil {
			clog.Error("World", "logUser", "Cant create user %s", err)
			return dat, err
		}
		storage.SaveUser(c.Name, dat)
		clog.Warn("World", "logUser", "Creating new user %s", dat)
	} else {
		clog.Test("", "", "%s", dat)
		err := json.Unmarshal(dat, &infos)
		if err != nil {
			clog.Error("World", "logUser", "Corrupted data for user %s : %s", c.Name, err)
			return dat, errors.New("ko")
		}
		clog.Info("World", "logUser", "Registering user %s", infos.ID)
	}

	infos.hubClient = c
	W.UserList[infos.ID] = infos
	W.Map.Entities[infos.X][infos.Y] = infos

	message := []byte(fmt.Sprintf("[BCST]%s", dat))
	W.AOIs.addEvent(infos.X, infos.Y, message)
	return dat, nil
}

func (W *WORLD) checkTargetHit(infos *USER) {
	var mobFound *MOB
	switch infos.Dir {
	case "up":
		for y := infos.Y - 1; y > infos.Y-infos.Pow; y-- {
			if W.Map.Entities[infos.X][y] != nil {
				mobFound = W.Map.Entities[infos.X][y].(*MOB)
				break
			}
		}
	case "down":
		for y := infos.Y + 1; y < infos.Y+infos.Pow; y++ {
			if W.Map.Entities[infos.X][y] != nil {
				mobFound = W.Map.Entities[infos.X][y].(*MOB)
				break
			}
		}
	case "left":
		for x := infos.X - 1; x > infos.X-infos.Pow; x-- {
			if W.Map.Entities[x][infos.Y] != nil {
				mobFound = W.Map.Entities[x][infos.Y].(*MOB)
				break
			}
		}
	case "right":
		for x := infos.X + 1; x < infos.X+infos.Pow; x++ {
			if W.Map.Entities[x][infos.Y] != nil {
				mobFound = W.Map.Entities[x][infos.Y].(*MOB)
				break
			}
		}
	}
	if mobFound != nil {
		message := []byte(fmt.Sprintf("[KILL]%s", mobFound.ID))
		W.AOIs.addEvent(mobFound.X, mobFound.Y, message)
		delete(W.MobList, mobFound.ID)
		W.Map.Entities[mobFound.X][mobFound.Y] = nil
	}
}

func (W *WORLD) CallToAction(c *hub.Client, cmd string, message []byte) {
	var infos USER
	err := json.Unmarshal(message, &infos)
	if err == nil {
		switch cmd {
		case "[FIRE]":
			W.checkTargetHit(&infos)
		case "[PMOV]":
			user := W.UserList[infos.ID]
			W.Map.Entities[user.X][user.Y] = nil
			user.X = infos.X
			user.Y = infos.Y
			W.Map.Entities[user.X][user.Y] = user
			mess := []byte(fmt.Sprintf("[BCST]%s", message))
			W.AOIs.addEvent(infos.X, infos.Y, mess)
		default:
			clog.Warn("World", "CallToAction", "Bad Action : %s", cmd)
		}
	} else {
		clog.Warn("World", "CallToAction", "%s", err)
	}
}

func (W *WORLD) sendWorldUpdate() {
	W.AOIs.computeUpdates()
	for _, player := range W.UserList {
		message, err := W.AOIs.getUpdateForPlayer(player.X, player.Y)
		if err == nil {
			mess := hub.NewMessage(nil, hub.ClientUser, player.hubClient, message)
			W.hub.Unicast <- mess
		}
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
			// start := time.Now()
			W.spawnMob()
			W.browseMob()
			if clog.LogLevel == 0 {
				W.Map.Draw()
			}
			W.sendWorldUpdate()

			// t := time.Now()
			// elapsed := t.Sub(start)
			// if elapsed >= timeStep {
			// 	clog.Error("", "", "Operations too long !!")
			// } else {
			// 	clog.Test("", "", "%c[HOperation last %s", 27, elapsed)
			// }
		default:
		}
	}
}

func (W *WORLD) GetMapArea(x, y int) []byte {
	return W.Map.ExportMapArea(x, y)
}

// func (W *WORLD) getShortPath(mob *MOB, user *USER) *pathfinder.Node {
// 	W.Graph = pathfinder.NewGraph(&W.Map, mob.X, mob.Y, user.X, user.Y)
// 	shortest_path := pathfinder.Astar(W.Graph)
// 	if len(shortest_path) > 0 {
// 		return shortest_path[1]
// 	} else {
// 		return nil
// 	}
// }
//
// func (W *WORLD) testPathFinder() {
// 	x := 50
// 	y := 11
// 	graph := pathfinder.NewGraph(&W.Map, 1, 1, x, y)
// 	shortest_path := pathfinder.Astar(graph)
// 	for _, path := range shortest_path {
// 		W.Map[path.X][path.Y] = -1
// 	}
// 	W.DrawMap()
// }

func Init(zeHub *hub.Hub) *WORLD {
	zeWorld := &WORLD{}
	zeWorld.MobList = make(map[string]*MOB)
	zeWorld.UserList = make(map[string]*USER)
	zeWorld.hub = zeHub
	zeWorld.Map = &MapData{}

	zeWorld.Map.loadTiledJSONMap("../data/final.json")

	zeWorld.AOIs = BuildAOIList(zeWorld)
	clog.Trace("", "", "%s", zeWorld.AOIs)

	// m := mapper.NewMap()
	// mapJSON, _ := json.Marshal(m)
	// clog.Trace("Mapper", "test", "%v", heightmap)
	// zeWorld.testPathFinder()
	// clog.Fatal("", "", nil)
	return zeWorld
}
