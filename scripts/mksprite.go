package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"io/ioutil"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	log.Println("Create sprites")
	if len(os.Args) != 3 {
		log.Fatal("illegal number of arguments")
	}

	srcDir := os.Args[1]
	dstBase := os.Args[2]

	err := mksprites(srcDir, dstBase)
	if err != nil {
		log.Fatal(err)
	}
}

func mksprites(srcDir, dstBase string) error {
	dstPath := dstBase + ".png"
	indexPath := dstBase + ".json"
	brushSize := 16 // width and height of individual brush images

	files, err := ioutil.ReadDir(srcDir)
	if err != nil {
		return err
	}

	size := 1
	for {
		capacity := size * size
		if capacity >= len(files) {
			break
		}
		size++
	}
	log.Printf("Spritesheet will have size %v for %v sprites", size, len(files))

	sideLen := size * brushSize
	sheet := image.NewRGBA(image.Rect(0, 0, sideLen, sideLen))
	lookup := make(map[string][]int)
	for i, f := range files {
		name := strings.TrimSuffix(f.Name(), ".png")

		path := filepath.Join(srcDir, f.Name())
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		defer src.Close()
		img, err := png.Decode(src)
		if err != nil {
			return err
		}

		r := img.Bounds()

		// check our assumptions about the brush size
		if r.Dx() != brushSize || r.Dy() != brushSize {
			return fmt.Errorf("unexpected brush size (%vx%v) for %q", r.Dx(), r.Dy(), path)
		}

		xOffset := i % size
		yOffset := int(math.Floor(float64(i / size)))
		x0 := xOffset * brushSize
		y0 := yOffset * brushSize
		x1 := x0 + brushSize
		y1 := y0 + brushSize

		rect := image.Rect(x0, y0, x1, y1)
		draw.Draw(sheet, rect, img, image.ZP, draw.Src)

		lookup[name] = []int{x0, y0, x1, y1}

		log.Printf("%v => %v,%v / %v,%v", name, x0, y0, x1, y1)
	}

	dst, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer dst.Close()
	err = png.Encode(dst, sheet)
	if err != nil {
		return err
	}

	index, err := os.Create(indexPath)
	if err != nil {
		return err
	}
	defer index.Close()
	err = json.NewEncoder(index).Encode(lookup)
	if err != nil {
		return err
	}

	return nil

}
