package tiled

import (
	"encoding/xml"
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	"image"
	"log"
	"os"

	_ "image/png"
)

const (
	assetDir                = "assets/"
	flippedHorizontallyFlag = 0x80000000
	flippedVerticallyFlag   = 0x40000000
	flippedDiagonallyFlag   = 0x20000000
)

type Map struct {
	Tiles map[int]*Tile // lookup tile image by gid

	Width      int             `xml:"width,attr"`
	Height     int             `xml:"height,attr"`
	TileWidth  int             `xml:"tilewidth,attr"`
	TileHeight int             `xml:"tileheight,attr"`
	TileSets   []*TileSetEntry `xml:"tileset"`
	Layer      *struct {
		Text   string `xml:",chardata"`
		ID     int    `xml:"id,attr"`
		Name   string `xml:"name,attr"`
		Width  int    `xml:"width,attr"`
		Height int    `xml:"height,attr"`
		Data   struct {
			Text string `xml:",chardata"`
		} `xml:"data"`
	} `xml:"layer"`
	ObjectGroups []*ObjectGroup `xml:"objectgroup"`
}

type TileSetEntry struct {
	Firstgid int    `xml:"firstgid,attr"`
	Source   string `xml:"source,attr"`
}

type ObjectGroup struct {
	Text    string    `xml:",chardata"`
	ID      string    `xml:"id,attr"`
	Name    string    `xml:"name,attr"`
	Opacity float64   `xml:"opacity,attr"`
	Objects []*Object `xml:"object"`
}

type Object struct {
	// Decoded from GID
	FlippedHorizontally, FlippedVertically, FlippedDiagonally bool

	Name     string  `xml:"name,attr"`
	ID       int     `xml:"id,attr"`
	GID      int     `xml:"gid,attr"`
	X        float64 `xml:"x,attr"`
	Y        float64 `xml:"y,attr"`
	Width    float64 `xml:"width,attr"`
	Height   float64 `xml:"height,attr"`
	Rotation float64 `xml:"rotation,attr"`
	Template string  `xml:"template,attr"`

	Properties []*Property `xml:"property"`
}

type Property struct {
	Name           string `xml:"name,attr"`
	Type           string `xml:"type,attr"`
	Value          string `xml:"value,attr"`
	InterpretedVal interface{}
}

type TileSet struct {
	Name  string  `xml:"name,attr"`
	Tiles []*Tile `xml:"tile"`
}

type Tile struct {
	ID    int   `xml:"id,attr"`
	Image Image `xml:"image"`
}

type Image struct {
	Data   *ebiten.Image
	Source string `xml:"source,attr"`
	Width  int    `xml:"width,attr"`
	Height int    `xml:"height,attr"`
}

type Template struct {
	TileSetEntry *TileSetEntry `xml:"tileset"`
	Object       *Object       `xml:"object"`
}

func NewMap(name string) *Map {
	f, err := os.Open(fmt.Sprintf("%v%v.tmx", assetDir, name))
	if err != nil {
		panic(err)
	}
	var map1 Map
	if err = xml.NewDecoder(f).Decode(&map1); err != nil {
		panic(err)
	}
	if err = f.Close(); err != nil {
		panic(err)
	}

	lookup := make(map[int]*Tile)

	for _, entry := range map1.TileSets {
		parseTileset(entry, lookup)
	}

	for _, group := range map1.ObjectGroups {
		for _, object := range group.Objects {
			// this seems silly, why not just use more attributes?
			object.FlippedHorizontally = (object.GID & flippedHorizontallyFlag) != 0
			object.FlippedVertically = (object.GID & flippedVerticallyFlag) != 0
			object.FlippedDiagonally = (object.GID & flippedDiagonallyFlag) != 0

			// clear the flags so the GID exists in the tile lookup
			object.GID = object.GID & ^flippedHorizontallyFlag
			object.GID = object.GID & ^flippedVerticallyFlag
			object.GID = object.GID & ^flippedDiagonallyFlag

			if object.Template != "" {
				// parse the template file
				f, err := os.Open(assetDir + object.Template)
				var template Template
				if err = xml.NewDecoder(f).Decode(&template); err != nil {
					panic(err)
				}
				if err = f.Close(); err != nil {
					panic(err)
				}

				// TODO load the tileSet? it's probably already loaded

				object.GID = template.Object.GID
				object.Width = template.Object.Width
				object.Height = template.Object.Height
				object.Properties = template.Object.Properties
			}
		}
	}

	map1.Tiles = lookup
	return &map1
}

func parseTileset(entry *TileSetEntry, lookup map[int]*Tile) {
	f, err := os.Open(assetDir + entry.Source)
	if err != nil {
		panic("File not found:" + err.Error())
	}
	var tileSet TileSet
	if err = xml.NewDecoder(f).Decode(&tileSet); err != nil {
		panic("Failed to decode: " + err.Error())
	}
	if err = f.Close(); err != nil {
		panic("Failed closing during decode: " + err.Error())
	}

	for _, tile := range tileSet.Tiles {
		if _, ok := lookup[entry.Firstgid+tile.ID]; ok {
			continue
		}
		name := assetDir + tile.Image.Source
		if f, err = os.Open(name); err != nil {
			panic(err)
		}
		img, _, err := image.Decode(f)
		if err != nil {
			log.Fatalln("failed decoding", name, "error:", err)
		}
		if err = f.Close(); err != nil {
			panic(err)
		}
		tile.Image.Data = ebiten.NewImageFromImage(img)
		lookup[entry.Firstgid+tile.ID] = tile
	}
}
