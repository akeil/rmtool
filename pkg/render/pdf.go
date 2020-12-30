package render

import (
	"bytes"
	"io"

	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"

	"akeil.net/akeil/rm"
	"akeil.net/akeil/rm/internal/logging"
)

const tsFormat = "2006-01-02 15:04:05"

// PDF renders all pages of the given document to a PDF file.
//
// The result is written to the given writer.
func PDF(d *rm.Document, w io.Writer) error {
	r := DefaultContext()
	return renderPDF(r, d, w)
}

func renderPDF(c *Context, d *rm.Document, w io.Writer) error {
	logging.Debug("Render PDF for document %q, type %q", d.ID(), d.FileType())
	pdf := setupPDF("A4", d)

	var err error
	if d.FileType() == rm.Pdf {
		err = overlayPDF(c, d, pdf)
	} else {
		err = drawingsPDF(c, pdf, d)
	}

	if err != nil {
		return err
	}
	return pdf.Output(w)
}

func drawingsPDF(c *Context, pdf *gofpdf.Fpdf, d *rm.Document) error {
	for i, pageID := range d.Pages() {
		err := doRenderPDFPage(c, pdf, d, pageID, i)
		if err != nil {
			return err
		}
	}

	return nil
}

// PDFPage renders a single drawing into a single one-page PDF.
func PDFPage(c *Context, d *rm.Document, pageID string, w io.Writer) error {
	pdf := setupPDF("A4", nil)

	err := doRenderPDFPage(c, pdf, d, pageID, 0)
	if err != nil {
		return err
	}

	return pdf.Output(w)
}

func setupPDF(pageSize string, d *rm.Document) *gofpdf.Fpdf {
	orientation := "P" // [P]ortrait or [L]andscape
	sizeUnit := "pt"
	fontDir := ""
	pdf := gofpdf.New(orientation, sizeUnit, pageSize, fontDir)

	//pdf.SetMargins(0, 8, 0) // left, top, right
	pdf.AliasNbPages("{totalPages}")
	pdf.SetFont("helvetica", "", 8)
	pdf.SetTextColor(127, 127, 127)
	pdf.SetProducer("rmtool", true)

	// If we are rendering a complete notebook, add metadata
	if d != nil {
		pdf.SetTitle(d.Name(), true)
		modified := d.LastModified().UTC()
		pdf.SetModificationDate(modified)
		pdf.SetCreationDate(modified)

		pdf.SetFooterFunc(func() {
			pdf.SetY(-20)
			pdf.SetX(24)
			pdf.Cellf(0, 10, "%d / {totalPages}  |  %v (v%d, %v)",
				pdf.PageNo(),
				d.Name(),
				d.Version(),
				d.LastModified().Local().Format(tsFormat))
		})
	}

	return pdf
}

func doRenderPDFPage(c *Context, pdf *gofpdf.Fpdf, doc *rm.Document, pageID string, i int) error {
	d, err := doc.Drawing(pageID)
	if err != nil {
		return err
	}

	// TODO: determine orientation, rotate image if neccessary
	// and set the page to Landscape
	pdf.AddPage()

	// TODO: add the background template

	return drawingToPDF(c, pdf, d)
}

// drawingToPDF renders the given Drawing to a bitmap and places it on the
// current page of the given PDF.
//
// This function is used to render a drawing onto an empty page
// AND to overlay an existing page with the drawing.
func drawingToPDF(c *Context, pdf *gofpdf.Fpdf, d *rm.Drawing) error {
	id := uuid.New().String()
	opts := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}

	// render to in-memory PNG
	var buf bytes.Buffer
	err := renderPNG(c, d, false, &buf)
	if err != nil {
		return err
	}
	// pdf.ImageOptions(...) will read frm the registered reader
	pdf.RegisterImageOptionsReader(id, opts, &buf)

	// The drawing will be scaled to the (usable) page width
	wPage, _ := pdf.GetPageSize()
	left, _, right, _ := pdf.GetMargins()
	w := wPage - left - right

	x := 0.0
	y := 0.0
	h := 0.0
	flow := false
	link := 0
	linkStr := ""
	pdf.ImageOptions(id, x, y, w, h, flow, opts, link, linkStr)

	return nil
}
