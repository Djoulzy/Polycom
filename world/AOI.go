package world

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/Djoulzy/Tools/clog"
)

type AOI struct {
	X         int
	Y         int
	Adjacents []*AOI
	Actions   [][]byte
	update    []byte
}

type AOIList [][]*AOI

func BuildAOIList(W *WORLD) *AOIList {
	width := W.Map.Width / AOIWidth
	height := W.Map.Height / AOIHeight
	clog.Info("", "", "Map Size: %dx%d -> AOIList: %dx%d", W.Map.Width, W.Map.Height, width, height)

	var L AOIList
	L = make(AOIList, width)
	for i := 0; i < width; i++ {
		L[i] = make([]*AOI, height)
		for j := 0; j < height; j++ {
			L[i][j] = &AOI{
				X: i,
				Y: j,
			}
		}
	}

	for i, cols := range L {
		for j, rows := range cols {
			for x := -1; x < 2; x++ {
				for y := -1; y < 2; y++ {
					dx := i + x
					dy := j + y
					if (x == 0 && y == 0) || dx < 0 || dy < 0 || dx >= width || dy >= height {
						continue
					}
					// clog.Trace("", "", "Adding %dx%d (%s) to %dx%d", dx, dy, L[dx][dy], i, j)
					rows.Adjacents = append(rows.Adjacents, L[dx][dy])
				}
			}
			// clog.Trace("", "", "%#v\n", rows.Adjacents)
		}
	}

	return &L
}

func (aoi AOI) String() string {
	return fmt.Sprintf("%dx%d", aoi.X, aoi.Y)
}

func (L AOIList) String() string {
	var str string
	for _, cols := range L {
		for _, rows := range cols {
			str = fmt.Sprintf("%s %s", str, rows)
		}
	}
	return str
}

func (L AOIList) getAOIfromCoord(x, y int) *AOI {
	AOIx := x / AOIWidth
	AOIy := y / AOIHeight
	return L[AOIx][AOIy]
}

func (aoi *AOI) addEvent(mess []byte) {
	for _, adj := range aoi.Adjacents {
		adj.Actions = append(adj.Actions, mess)
	}
	aoi.Actions = append(aoi.Actions, mess)
}

func (L *AOIList) addEvent(x, y int, mess []byte) {
	aoi := L.getAOIfromCoord(x, y)
	aoi.addEvent(mess)
}

func (L AOIList) computeUpdates() {
	for _, cols := range L {
		for _, aoi := range cols {
			aoi.update = bytes.Join(aoi.Actions, []byte("|"))
			aoi.Actions = aoi.Actions[:0]
		}
	}
}

func (L *AOIList) getUpdateForPlayer(x, y int) ([]byte, error) {
	aoi := L.getAOIfromCoord(x, y)
	if len(aoi.update) > 0 {
		return aoi.update, nil
	} else {
		return aoi.update, errors.New("No update")
	}
}
