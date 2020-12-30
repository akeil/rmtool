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
	"akeil.net/akeil/rm/internal/imaging"
	"akeil.net/akeil/rm/internal/logging"
)

var brushNames = map[rm.BrushType]string{
	rm.Ballpoint:          "ballpoint",
	rm.BallpointV5:        "ballpoint",
	rm.Pencil:             "pencil",
	rm.PencilV5:           "pencil",
	rm.MechanicalPencil:   "mech-pencil",
	rm.MechanicalPencilV5: "mech-pencil",
	rm.Marker:             "marker",
	rm.MarkerV5:           "marker",
	rm.Fineliner:          "fineliner",
	rm.FinelinerV5:        "fineliner",
	rm.Highlighter:        "highlighter",
	rm.HighlighterV5:      "highlighter",
	rm.PaintBrush:         "ballpoint", // TODO add mask image and change name
	rm.PaintBrushV5:       "ballpoint", // TODO add mask image and change name
	rm.CalligraphyV5:      "ballpoint", // TODO add mask image and change name
}

var defaultColors = map[rm.BrushColor]color.Color{
	rm.Black: color.Black,
	rm.Gray:  color.RGBA{127, 127, 127, 255},
	rm.White: color.White,
}

// Context holds parameters and cached data for rendering operations.
//
// If multiple drawings are rendered, they should use the same Context.
type Context struct {
	DataDir     string
	colors      map[rm.BrushColor]color.Color
	sprites     *image.RGBA
	spriteIndex map[string][]int
	spriteMx    sync.Mutex
	tpl         map[string]template
	tplCache    map[string]image.Image
	tplMx       sync.Mutex
}

// NewContext sets up a new rendering context.
//
// dataDir should point to a directory with a spritesheet for the brushes
// and a subdirectory 'templates' with page backgrounds.
func NewContext(dataDir string) *Context {
	return &Context{
		DataDir: "data",
	}
}

func DefaultContext() *Context {
	// TODO hardcoded path - choose a more sensible value
	return NewContext("./data")
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
	lookup := c.colors
	if lookup == nil {
		lookup = defaultColors
	}
	col := lookup[bc]
	if col == nil {
		return nil, fmt.Errorf("invalid color %v", bc)
	}

	name := brushNames[bt]
	if name == "" {
		return nil, fmt.Errorf("unsupported brush type %v", bt)
	}

	i, err := c.loadBrushMask(name)
	if err != nil {
		return nil, err
	}
	mask := imaging.CreateMask(i)

	switch bt {
	case rm.Ballpoint, rm.BallpointV5:
		return &Ballpoint{
			mask:  mask,
			fill:  image.NewUniform(col),
			color: col,
		}, nil
	case rm.Pencil, rm.PencilV5:
		return &Pencil{
			mask: mask,
			fill: image.NewUniform(col),
		}, nil
	case rm.MechanicalPencil, rm.MechanicalPencilV5:
		return &MechanicalPencil{
			mask: mask,
			fill: image.NewUniform(col),
		}, nil
	case rm.Marker, rm.MarkerV5:
		return &Marker{
			mask: mask,
			fill: image.NewUniform(col),
		}, nil
	case rm.Fineliner, rm.FinelinerV5:
		return &Fineliner{
			mask:  mask,
			fill:  image.NewUniform(col),
			color: col,
		}, nil
	case rm.Highlighter, rm.HighlighterV5:
		return &Highlighter{
			mask: mask,
			fill: image.NewUniform(col),
		}, nil
	case rm.PaintBrush, rm.PaintBrushV5:
		return &Paintbrush{
			fill: image.NewUniform(col),
		}, nil
	default:
		logging.Warning("unsupported brush type %v", bt)
		return loadBasePen(mask, col), nil
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

	// sanity check
	if rect.Dx() > c.sprites.Bounds().Dx() || rect.Dy() > c.sprites.Bounds().Dy() {
		return nil, fmt.Errorf("sprite bounds not within spritesheet dimensions")
	}

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
	pj := filepath.Join(c.DataDir, "sprites.json")
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
	img, err := readPNG(c.DataDir, "sprites.png")
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

func (c *Context) loadTemplate(name string) (image.Image, error) {
	c.tplMx.Lock()
	defer c.tplMx.Unlock()
	if c.tplCache == nil {
		c.tplCache = make(map[string]image.Image)
	}
	cached := c.tplCache[name]
	if cached != nil {
		return cached, nil
	}

	/*
		TODO: apparently, we do not need the index
			  filename is directly contained in pagedata
			  Maybe 'orientation' is important?

		err := c.lazyLoadTemplateIndex()
		if err != nil {
			return nil, err
		}

		t, ok := c.tpl[name]
		if !ok {
			return nil, fmt.Errorf("no template file found for %q", name)
		}
	*/
	img, err := readPNG(c.DataDir, "templates", name+".png")
	if err != nil {
		return nil, err
	}

	c.tplCache[name] = img

	return img, nil
}

func (c *Context) lazyLoadTemplateIndex() error {
	if c.tpl != nil {
		return nil
	}

	p := filepath.Join(c.DataDir, "templates", "templates.json")
	logging.Debug("Load template index from %q", p)
	f, err := os.Open(p)
	if err != nil {
		return err
	}
	defer f.Close()

	var dst map[string][]template

	err = json.NewDecoder(f).Decode(&dst)
	if err != nil {
		return err
	}

	c.tpl = make(map[string]template)
	data := dst["templates"]
	if data == nil {
		return fmt.Errorf("unexpected JSON in %q - missing 'templates' member", p)
	}
	for _, t := range data {
		c.tpl[t.Name] = t
	}

	return nil
}

type template struct {
	Name      string `json:"name"`
	Filename  string `json:"filename"`
	Landscape bool   `json:"Landscape"`
}

func readPNG(path ...string) (image.Image, error) {
	p := filepath.Join(path...)
	logging.Debug("Read PNG image from %q", p)

	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return png.Decode(f)
}
