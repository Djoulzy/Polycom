package world

import (
	"encoding/json"
	"fmt"
	"io/ioutil"

	"github.com/Djoulzy/Tools/clog"
)

type FILELAYER struct {
	Data    []int  `bson:"data" json:"data"`
	Name    string `bson:"name" json:"name"`
	Width   int    `bson:"width" json:"width"`
	Height  int    `bson:"height" json:"height"`
	Opacity int    `bson:"opacity" json:"opacity"`
	Type    string `bson:"type" json:"type"`
	Visible bool   `bson:"visible" json:"visible"`
	X       int    `bson:"x" json:"x"`
	Y       int    `bson:"y" json:"y"`
	Offsetx int    `bson:"offsetx" json:"offsetx"`
	Offsety int    `bson:"offsety" json:"offsety"`
}

type FILETILESET struct {
	Columns     int    `bson:"columns" json:"columns"`
	Firstgid    int    `bson:"firstgid" json:"firstgid"`
	Image       string `bson:"image" json:"image"`
	Imageheight int    `bson:"imageheight" json:"imageheight"`
	Imagewidth  int    `bson:"imagewidth" json:"imagewidth"`
	Margin      int    `bson:"margin" json:"margin"`
	Name        string `bson:"name" json:"name"`
	Spacing     int    `bson:"spacing" json:"spacing"`
	Tilecount   int    `bson:"tilecount" json:"tilecount"`
	Tileheight  int    `bson:"tileheight" json:"tileheight"`
	Tilewidth   int    `bson:"tilewidth" json:"tilewidth"`
}

type FILEMAP struct {
	Width        int           `bson:"width" json:"width"`
	Height       int           `bson:"height" json:"height"`
	Nextobjectid int           `bson:"nextobjectid" json:"nextobjectid"`
	Orientation  string        `bson:"orientation" json:"orientation"`
	Renderorder  string        `bson:"renderorder" json:"renderorder"`
	Tiledversion string        `bson:"tiledversion" json:"tiledversion"`
	Tileheight   int           `bson:"tileheight" json:"tileheight"`
	Tilewidth    int           `bson:"tilewidth" json:"tilewidth"`
	Type         string        `bson:"type" json:"type"`
	Version      int           `bson:"version" json:"version"`
	Layers       []FILELAYER   `bson:"layers" json:"layers"`
	Tilesets     []FILETILESET `bson:"tilesets" json:"tilesets"`
}

type MapData struct {
	Width    int
	Height   int
	Entities [][]interface{}
	Ground   [][]int
	Block    [][]int
	FileData FILEMAP
}

func (M *MapData) loadTiledJSONMap(file string) {
	dat, _ := ioutil.ReadFile(file)
	err := json.Unmarshal(dat, &M.FileData)
	if err != nil {
		clog.Error("", "", "%s", err)
	}

	// M.FileData := mapper.NewMap()
	M.Width = M.FileData.Layers[0].Width
	M.Height = M.FileData.Layers[0].Height

	M.Ground = make([][]int, M.Width)
	M.Block = make([][]int, M.Width)
	M.Entities = make([][]interface{}, M.Width)
	for i := 0; i < M.Width; i++ {
		M.Entities[i] = make([]interface{}, M.Height)
		M.Ground[i] = make([]int, M.Height)
		M.Block[i] = make([]int, M.Height)
	}

	row := 0
	for row < M.Height {
		col := 0
		for col < M.Width {
			M.Ground[col][row] = M.FileData.Layers[0].Data[(row*M.Width)+col]
			M.Block[col][row] = M.FileData.Layers[1].Data[(row*M.Width)+col]
			M.Entities[col][row] = nil
			col++
		}
		row++
	}
}

func (M *MapData) ExportMapArea(x, y int) []byte {
	var startx, starty int
	tmp := M.FileData

	if x-1 < 0 {
		startx = 0
		tmp.Width = AOIWidth * 2
	} else {
		startx = (x - 1) * AOIWidth
		tmp.Width = AOIWidth * 3
	}

	if y-1 < 0 {
		starty = 0
		tmp.Height = AOIHeight * 2
	} else {
		starty = (y - 1) * AOIHeight
		tmp.Height = AOIHeight * 3
	}

	tmp.Layers[0].Data = make([]int, tmp.Width*tmp.Height)
	tmp.Layers[1].Data = make([]int, tmp.Width*tmp.Height)

	cpt := 0
	for j := starty; j < starty+tmp.Height; j++ {
		for i := startx; i < startx+tmp.Width; i++ {
			tmp.Layers[0].Data[cpt] = M.Ground[i][j]
			tmp.Layers[1].Data[cpt] = M.Block[i][j]
			cpt++
		}
	}

	tmp.Layers[0].Width = tmp.Width
	tmp.Layers[0].Height = tmp.Height
	// tmp.Layers[0].X = startx * tmp.Tilewidth
	// tmp.Layers[0].Y = starty * tmp.Tileheight
	// tmp.Layers[0].Offsetx = 0
	// tmp.Layers[0].Offsety = 0
	tmp.Layers[0].X = 0
	tmp.Layers[0].Y = 0
	tmp.Layers[0].Offsetx = startx * tmp.Tilewidth
	tmp.Layers[0].Offsety = starty * tmp.Tileheight

	tmp.Layers[1].Width = tmp.Width
	tmp.Layers[1].Height = tmp.Height
	// tmp.Layers[1].X = startx * tmp.Tilewidth
	// tmp.Layers[1].Y = starty * tmp.Tileheight
	// tmp.Layers[1].Offsetx = 0
	// tmp.Layers[1].Offsety = 0
	tmp.Layers[1].X = 0
	tmp.Layers[1].Y = 0
	tmp.Layers[1].Offsetx = startx * tmp.Tilewidth
	tmp.Layers[1].Offsety = starty * tmp.Tileheight

	json, _ := json.MarshalIndent(tmp, "", "    ")
	return json
}

func (M *MapData) Draw() {
	fmt.Printf("%c[H", 27)
	visuel := ""
	display := "*"
	for y := 0; y < M.Height; y++ {
		for x := 0; x < M.Width; x++ {
			val := M.Block[x][y]
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
			if M.Entities[x][y] != nil {
				switch M.Entities[x][y].(type) {
				case *MOB:
					visuel = clog.GetColoredString(" Z ", "white", "red")
				case *USER:
					visuel = clog.GetColoredString(" P ", "black", "yellow")
				}
			}
			display = fmt.Sprintf("%s%s", display, visuel)
		}
		display = fmt.Sprintf("%s*\n*", display)
	}
	fmt.Printf("%s", display)
}
