package render

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"os"
	"path/filepath"
	"sync"

	"akeil.net/akeil/rm"
	"akeil.net/akeil/rm/internal/logging"
)

type Context struct {
	dataDir     string
	colors      map[rm.BrushColor]color.Color
	sprites     *image.RGBA
	spriteIndex map[string][]int
	spriteMx    sync.Mutex
}

func NewContext(dataDir string) *Context {
	var colors = map[rm.BrushColor]color.Color{
		rm.Black: color.Black,
		rm.Gray:  color.RGBA{127, 127, 127, 255},
		rm.White: color.White,
	}
	return &Context{
		dataDir: "data",
		colors:  colors,
	}
}

// Page draws a single page to a PNG and writes it to the given writer.
func (c *Context) Page(doc *rm.Document, pageID string, w io.Writer) error {
	return renderPage(c, doc, pageID, w)
}

// PDF renders all pages from a document to a PDF file.
//
// The resulting PDF document is written to the given writer.
func (c *Context) PDF(doc *rm.Document, w io.Writer) error {
	return renderPDF(c, doc, w)
}

func (c *Context) loadBrush(bt rm.BrushType, bc rm.BrushColor) (Brush, error) {
	col := c.colors[bc]
	if col == nil {
		return nil, fmt.Errorf("invalid color %v", bc)
	}

	switch bt {
	case rm.Ballpoint, rm.BallpointV5:
		return loadBallpoint(c, col)
	case rm.Pencil, rm.PencilV5:
		return loadPencil(c, col)
	case rm.MechanicalPencil, rm.MechanicalPencilV5:
		return loadMechanicalPencil(c, col)
	case rm.Marker, rm.MarkerV5:
		return loadMarker(c, col)
	case rm.Fineliner, rm.FinelinerV5:
		return loadFineliner(c, col)
	case rm.Highlighter, rm.HighlighterV5:
		return loadHighlighter(c, col)
	case rm.PaintBrush, rm.PaintBrushV5:
		return loadPaintbrush(c, col)
	default:
		logging.Warning("unsupported brush type %v", bt)
		return loadBasePen(c, col)
	}
}

// loadBrushMask loads a brush image identified by name.
func (c *Context) loadBrushMask(name string) (image.Image, error) {
	err := c.lazyLoadSpritesheet()
	if err != nil {
		return nil, err
	}

	idx := c.spriteIndex[name]
	if idx == nil {
		return nil, fmt.Errorf("no sprite image for brush %q", name)
	} else if len(idx) != 4 {
		return nil, fmt.Errorf("invalid sprite entry for brush %q", name)
	}

	rect := image.Rect(idx[0], idx[1], idx[2], idx[3])
	// TODO: check bounds?
	return c.sprites.SubImage(rect), nil
}

func (c *Context) lazyLoadSpritesheet() error {
	c.spriteMx.Lock()
	defer c.spriteMx.Unlock()
	if c.sprites != nil {
		// already loaded
		return nil
	}

	// index map
	pj := filepath.Join(c.dataDir, "sprites.json")
	logging.Debug("Load sprite index from %q", pj)
	j, err := os.Open(pj)
	if err != nil {
		return err
	}
	defer j.Close()
	err = json.NewDecoder(j).Decode(&c.spriteIndex)
	if err != nil {
		return err
	}

	// image
	pi := filepath.Join(c.dataDir, "sprites.png")
	logging.Debug("Load spritesheet from %q", pi)
	i, err := os.Open(pi)
	if err != nil {
		return err
	}
	defer i.Close()
	img, err := png.Decode(i)
	if err != nil {
		return err
	}

	// make the image an RGBA (allows SubImage(...)
	c.sprites = image.NewRGBA(img.Bounds())
	for x := 0; x < c.sprites.Bounds().Dx(); x++ {
		for y := 0; y < c.sprites.Bounds().Dy(); y++ {
			c.sprites.Set(x, y, img.At(x, y))
		}
	}

	return nil
}
