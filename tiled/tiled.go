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
	Tiles map[int]*ebiten.Image // lookup tile image by gid

	Width      int `xml:"width,attr"`
	Height     int `xml:"height,attr"`
	TileWidth  int `xml:"tilewidth,attr"`
	TileHeight int `xml:"tileheight,attr"`
	TileSets   []*struct {
		Text     string `xml:",chardata"`
		Firstgid int    `xml:"firstgid,attr"`
		Source   string `xml:"source,attr"`
	} `xml:"tileset"`
	Layer *struct {
		Text   string `xml:",chardata"`
		ID     int `xml:"id,attr"`
		Name   string `xml:"name,attr"`
		Width  int `xml:"width,attr"`
		Height int `xml:"height,attr"`
		Data   struct {
			Text     string `xml:",chardata"`
		} `xml:"data"`
	} `xml:"layer"`
	ObjectGroups []*ObjectGroup `xml:"objectgroup"`
}

type ObjectGroup struct {
	Text    string `xml:",chardata"`
	ID      string `xml:"id,attr"`
	Name    string `xml:"name,attr"`
	Objects []*struct {
		// Decoded from GID
		FlippedHorizontally, FlippedVertically, FlippedDiagonally bool

		Text   string `xml:",chardata"`
		ID     int `xml:"id,attr"`
		GID    int `xml:"gid,attr"`
		X      float64 `xml:"x,attr"`
		Y      float64 `xml:"y,attr"`
		Width  float64 `xml:"width,attr"`
		Height float64 `xml:"height,attr"`
	} `xml:"object"`
}

type TileSet struct {
	Name  string `xml:"name,attr"`
	Tiles []*struct {
		ID int `xml:"id,attr"`
		Image struct {
			Source string `xml:"source,attr"`
			Width int `xml:"width,attr"`
			Height int `xml:"height,attr"`
		} `xml:"image"`
		ObjectGroup *ObjectGroup `xml:"objectgroup"`
	} `xml:"tile"`
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
		}
	}

	lookup := make(map[int]*ebiten.Image)

	for _, entry := range map1.TileSets {
		f, err = os.Open(assetDir + entry.Source)
		var tileSet TileSet
		if err = xml.NewDecoder(f).Decode(&tileSet); err != nil {
			panic(err)
		}
		if err = f.Close(); err != nil {
			panic(err)
		}

		for _, tile := range tileSet.Tiles {
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
			tilesImage := ebiten.NewImageFromImage(img)
			lookup[entry.Firstgid+tile.ID] = tilesImage
		}
	}

	map1.Tiles = lookup
	return &map1
}
