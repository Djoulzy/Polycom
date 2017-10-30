package world

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/Djoulzy/Polycom/hub"
	"github.com/Djoulzy/Polycom/world/mapper"
	"github.com/Djoulzy/Polycom/world/pathfinder"
	"github.com/Djoulzy/Tools/clog"
	"github.com/nu7hatch/gouuid"
)

const (
	timeStep  = 100 * time.Millisecond // Actualisation 10 par seconde
	tileSize  = 32
	mobSpeed  = 8
	maxMobNum = 100
)

type MOB struct {
	ID        string `bson:"id" json:"id"`
	Type      string `bson:"type" json:"type"`
	Face      string `bson:"face" json:"face"`
	ComID     int    `bson:"num" json:"num"`
	Dir       string `bson:"move" json:"move"`
	X         int    `bson:"x" json:"x"` // Col nums
	Y         int    `bson:"y" json:"y"` // Row nums
	Pow       int    `bson:"pow" json:"pow"`
	Speed     int    `bson:"speed" json:"speed"`
	waitState int
}

type TILE struct {
	Type int
	ID   string
}

type WORLD struct {
	hub      *hub.Hub
	MobList  map[string]*MOB
	UserList map[string]*MOB
	Width    int
	Height   int
	Map      pathfinder.MapData
	UserMap  [][]*MOB
	Graph    *pathfinder.Graph
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

func (W *WORLD) findSpawnPlace() (int, int) {
	for {
		x := rand.Intn(W.Width)
		y := rand.Intn(W.Height)
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
			ID:        uid.String(),
			Type:      "M",
			Face:      face,
			ComID:     1,
			Speed:     mobSpeed,
			waitState: 0,
		}
		mob.X, mob.Y = W.findSpawnPlace()
		W.UserMap[mob.X][mob.Y] = mob
		W.MobList[mob.ID] = mob
		message := []byte(fmt.Sprintf("[NMOB]%s", mob.ID))
		clog.Info("WORLD", "spawnMob", "Spawning new mob %s", mob.ID)
		mess := hub.NewMessage(nil, hub.ClientUser, nil, message)
		W.hub.Broadcast <- mess
	}
}

func (W *WORLD) findCloserUser(mob *MOB) (*MOB, error) {
	var distFound float64 = 0
	var userFound *MOB = nil
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
	mess := hub.NewMessage(nil, hub.ClientUser, nil, message)
	W.hub.Broadcast <- mess
	mob.waitState = mob.Speed
}

func (W *WORLD) tileIsFree(x, y int) bool {
	if W.Map[x][y] == 0 && W.UserMap[x][y] == nil {
		return true
	}
	return false
}

func (W *WORLD) moveSIMPLE(mob *MOB, prey *MOB) {
	clog.Info("World", "moveMob", "Seeking for %s", prey.ID)
	// if math.Abs(float64(prey.X-mob.X)) < math.Abs(float64(prey.Y-mob.Y)) {
	if mob.Y > prey.Y && W.tileIsFree(mob.X, mob.Y-1) {
		W.UserMap[mob.X][mob.Y] = nil
		mob.Y -= 1
		mob.Dir = "up"
		W.sendMobPos(mob)
		W.UserMap[mob.X][mob.Y] = mob
		return
	}
	if mob.Y < prey.Y && W.tileIsFree(mob.X, mob.Y+1) {
		W.UserMap[mob.X][mob.Y] = nil
		mob.Y += 1
		mob.Dir = "down"
		W.sendMobPos(mob)
		W.UserMap[mob.X][mob.Y] = mob
		return
	}
	if mob.X > prey.X && W.tileIsFree(mob.X-1, mob.Y) {
		W.UserMap[mob.X][mob.Y] = nil
		mob.X -= 1
		mob.Dir = "left"
		W.sendMobPos(mob)
		W.UserMap[mob.X][mob.Y] = mob
		return
	}
	if mob.X < prey.X && W.tileIsFree(mob.X+1, mob.Y) {
		W.UserMap[mob.X][mob.Y] = nil
		mob.X += 1
		mob.Dir = "right"
		W.sendMobPos(mob)
		W.UserMap[mob.X][mob.Y] = mob
		return
	}
}

func (W *WORLD) moveASTAR(mob *MOB, prey *MOB) {
	node := W.getShortPath(mob, prey)
	if node != nil {
		clog.Info("World", "moveMob", "Seeking for %s", prey.ID)
		if node.X > mob.X {
			mob.Dir = "right"
		} else if node.X < mob.X {
			mob.Dir = "left"
		} else if node.Y < mob.Y {
			mob.Dir = "up"
		} else if node.Y > mob.Y {
			mob.Dir = "down"
		}
		mob.X = node.X
		mob.Y = node.Y
		W.sendMobPos(mob)
	}
}

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

func (W *WORLD) logUser(infos *MOB) {
	if W.UserList[infos.ID] == nil {
		W.UserList[infos.ID] = infos
		W.UserMap[infos.X][infos.Y] = infos
	}
}

func (W *WORLD) checkTargetHit(infos *MOB) {
	var mobFound *MOB
	switch infos.Dir {
	case "up":
		for y := infos.Y - 1; y > infos.Y-infos.Pow; y-- {
			if W.UserMap[infos.X][y] != nil {
				mobFound = W.UserMap[infos.X][y]
				break
			}
		}
	case "down":
		for y := infos.Y + 1; y < infos.Y+infos.Pow; y++ {
			if W.UserMap[infos.X][y] != nil {
				mobFound = W.UserMap[infos.X][y]
				break
			}
		}
	case "left":
		for x := infos.X - 1; x > infos.X-infos.Pow; x-- {
			if W.UserMap[x][infos.Y] != nil {
				mobFound = W.UserMap[x][infos.Y]
				break
			}
		}
	case "right":
		for x := infos.X + 1; x < infos.X+infos.Pow; x++ {
			if W.UserMap[x][infos.Y] != nil {
				mobFound = W.UserMap[x][infos.Y]
				break
			}
		}
	}
	if mobFound != nil {
		message := []byte(fmt.Sprintf("[KILL]%s", mobFound.ID))
		mess := hub.NewMessage(nil, hub.ClientUser, nil, message)
		W.hub.Broadcast <- mess
		delete(W.MobList, mobFound.ID)
		W.UserMap[mobFound.X][mobFound.Y] = nil
	}
}

func (W *WORLD) CallToAction(cmd string, message []byte) {
	var infos MOB
	err := json.Unmarshal(message, &infos)
	if err == nil {
		switch cmd {
		case "[FIRE]":
			W.checkTargetHit(&infos)
		case "[BCST]":
			if (infos.Type == "P") && (W.UserList[infos.ID]) == nil {
				clog.Warn("World", "CallToAction", "Registering user %s", infos.ID)
				W.UserList[infos.ID] = &infos
				W.UserMap[infos.X][infos.Y] = &infos
			} else {
				user := W.UserList[infos.ID]
				W.UserMap[user.X][user.Y] = nil
				user.X = infos.X
				user.Y = infos.Y
				W.UserMap[user.X][user.Y] = user
			}
		}
	} else {
		clog.Warn("World", "CallToAction", "%s", err)
	}
}

func (W *WORLD) DrawMap() {
	fmt.Printf("%c[H", 27)
	visuel := ""
	display := "*"
	for y := 0; y < W.Height; y++ {
		for x := 0; x < W.Width; x++ {
			val := W.Map[x][y]
			if val == 0 {
				visuel = "   "
			} else if val == -1 {
				visuel = clog.GetColoredString(" + ", "black", "green")
			} else if val == 1000 {
				visuel = clog.GetColoredString(" D ", "black", "yellow")
			} else if val == 2000 {
				visuel = clog.GetColoredString(" F ", "white", "blue")
			} else {
				visuel = clog.GetColoredString(" X ", "white", "white")
			}
			User := W.UserMap[x][y]
			if User != nil {
				if User.Type == "M" {
					visuel = clog.GetColoredString(" Z ", "white", "red")
				} else if User.Type == "P" {
					visuel = clog.GetColoredString(" P ", "black", "yellow")
				}
			}
			// for _, mob := range W.MobList {
			// 	if mob.X == x && mob.Y == y {
			// 		visuel = clog.GetColoredString(" Z ", "white", "red")
			// 		break
			// 	}
			// }
			// for _, user := range W.UserList {
			// 	if user.X == x && user.Y == y {
			// 		visuel = clog.GetColoredString(" P ", "black", "yellow")
			// 		break
			// 	}
			// }
			display = fmt.Sprintf("%s%s", display, visuel)
		}
		display = fmt.Sprintf("%s*\n*", display)
	}
	fmt.Printf("%s", display)
}

func (W *WORLD) Run() {
	ticker := time.NewTicker(timeStep)
	defer func() {
		ticker.Stop()
	}()

	for {
		select {
		case <-ticker.C:
			start := time.Now()
			W.spawnMob()
			W.browseMob()
			if clog.LogLevel == 0 {
				W.DrawMap()
			}
			t := time.Now()
			elapsed := t.Sub(start)
			if elapsed >= timeStep {
				clog.Error("", "", "Operations too long !!")
			} else {
				clog.Test("", "", "Operation last %s", elapsed)
			}
		default:
		}
	}
}

func (W *WORLD) loadMap(file string) {
	// var zemap FILEMAP
	// dat, _ := ioutil.ReadFile(file)
	// err := json.Unmarshal(dat, &zemap)
	// if err != nil {
	// 	clog.Error("", "", "%s", err)
	// }
	zemap := mapper.NewMap()
	W.Width = zemap.Layers[2].Width
	W.Height = zemap.Layers[2].Height

	W.Map = make(pathfinder.MapData, W.Width)
	W.UserMap = make([][]*MOB, W.Width)
	for i := 0; i < W.Width; i++ {
		W.UserMap[i] = make([]*MOB, W.Height)
		W.Map[i] = make([]int, W.Height)
	}

	row := 0
	for row < W.Height {
		col := 0
		for col < W.Width {
			W.Map[col][row] = zemap.Layers[2].Data[(row*W.Width)+col]
			W.UserMap[col][row] = nil
			col++
		}
		row++
	}
}

func (W *WORLD) getShortPath(mob *MOB, user *MOB) *pathfinder.Node {
	W.Graph = pathfinder.NewGraph(&W.Map, mob.X, mob.Y, user.X, user.Y)
	shortest_path := pathfinder.Astar(W.Graph)
	if len(shortest_path) > 0 {
		return shortest_path[1]
	} else {
		return nil
	}
}

func (W *WORLD) testPathFinder() {
	x := 50
	y := 11
	graph := pathfinder.NewGraph(&W.Map, 1, 1, x, y)
	shortest_path := pathfinder.Astar(graph)
	for _, path := range shortest_path {
		W.Map[path.X][path.Y] = -1
	}
	W.DrawMap()
}

func Init(zeHub *hub.Hub) *WORLD {
	zeWorld := &WORLD{}
	zeWorld.MobList = make(map[string]*MOB)
	zeWorld.UserList = make(map[string]*MOB)
	zeWorld.hub = zeHub

	zeWorld.loadMap("../data/zone1.json")

	// m := mapper.NewMap()
	// mapJSON, _ := json.Marshal(m)
	// clog.Trace("Mapper", "test", "%v", heightmap)
	// zeWorld.testPathFinder()
	// clog.Fatal("", "", nil)
	return zeWorld
}
