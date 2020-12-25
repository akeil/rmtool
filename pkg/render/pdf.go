package render

import (
	"bytes"
	"io"

	"github.com/google/uuid"
	"github.com/jung-kurt/gofpdf"

	"akeil.net/akeil/rm"
	"akeil.net/akeil/rm/internal/logging"
)

func RenderPDF(d *rm.Document, w io.Writer) error {
	logging.Debug("Render PDF for document %q, type %q", d.ID(), d.FileType())
	pdf := setupPDF("A4", d)

	var err error
	if d.FileType() == rm.Pdf {
		err = overlayPDF(d, pdf)
	} else {
		err = renderDrawingsPDF(pdf, d)
	}

	if err != nil {
		return err
	}
	return pdf.Output(w)
}

func renderDrawingsPDF(pdf *gofpdf.Fpdf, d *rm.Document) error {
	for i, pageId := range d.Pages() {
		// TODO: insert a blank page if there is no drawing
		err := doRenderPDFPage(pdf, d, pageId, i)
		if err != nil {
			return err
		}
	}

	return nil
}

func RenderPDFPage(d *rm.Document, pageId string, w io.Writer) error {
	pdf := setupPDF("A4", nil)

	err := doRenderPDFPage(pdf, d, pageId, 0)
	if err != nil {
		return err
	}

	return pdf.Output(w)
}

const tsFormat = "2006-01-02 15:04:05"

func setupPDF(pageSize string, d *rm.Document) *gofpdf.Fpdf {
	orientation := "P" // [P]ortrait or [L]andscape
	sizeUnit := "pt"
	fontDir := ""
	pdf := gofpdf.New(orientation, sizeUnit, pageSize, fontDir)

	pdf.SetMargins(0, 8, 0) // left, top, right
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

func doRenderPDFPage(pdf *gofpdf.Fpdf, doc *rm.Document, pageId string, i int) error {
	d, err := doc.Drawing(pageId)
	if err != nil {
		return err
	}

	// TODO: determine orientation, rotate image if neccessary
	// and set the page to Landscape
	pdf.AddPage()

	// TODO: add the background template

	return renderDrawingToPDF(pdf, d)
}

func renderDrawingToPDF(pdf *gofpdf.Fpdf, d *rm.Drawing) error {
	name := uuid.New().String()
	opts := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}

	// render to PNG
	var buf bytes.Buffer
	err := RenderPNG(d, &buf)
	if err != nil {
		return err
	}
	pdf.RegisterImageOptionsReader(name, opts, &buf)

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
	pdf.ImageOptions(name, x, y, w, h, flow, opts, link, linkStr)

	return nil
}
