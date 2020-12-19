package render

import (
	"bytes"
	"fmt"
	"io"

	"akeil.net/akeil/rm"
	"github.com/jung-kurt/gofpdf"
)

func RenderPDF(n *rm.Notebook, w io.Writer) error {
	pdf := setupPDF("A4", n)

	for i, p := range n.Pages {
		// TODO: insert a blank page if there is no drawing
		if p.HasDrawing() {
			err := doRenderPDFPage(pdf, p, i)
			if err != nil {
				return err
			}
		}
	}

	return pdf.Output(w)
}

func RenderPDFPage(p *rm.Page, w io.Writer) error {
	pdf := setupPDF("A4", nil)

	err := doRenderPDFPage(pdf, p, 0)
	if err != nil {
		return err
	}

	return pdf.Output(w)
}

const tsFormat = "2006-01-02 15:04:05"

func setupPDF(pageSize string, n *rm.Notebook) *gofpdf.Fpdf {
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
	if n != nil {
		title := n.Meta.VisibleName
		pdf.SetTitle(title, true)
		// TODO: set from metadata?
		modified := n.Meta.LastModified.UTC()
		pdf.SetModificationDate(modified)
		pdf.SetCreationDate(modified)

		pdf.SetFooterFunc(func() {
			pdf.SetY(-20)
			pdf.SetX(24)
			pdf.Cellf(0, 10, "%d / {totalPages}  |  %v (v%d, %v)",
				pdf.PageNo(),
				title,
				n.Meta.Version,
				n.Meta.LastModified.Local().Format(tsFormat))
		})
	}

	return pdf
}

func doRenderPDFPage(pdf *gofpdf.Fpdf, p *rm.Page, i int) error {
	// TODO: determine orientation, rotate image if neccessary
	// and set the page to Landscape
	pdf.AddPage()

	// TODO: add the background template

	name := fmt.Sprintf("drawing-%d", i)
	opts := gofpdf.ImageOptions{ImageType: "PNG", ReadDpi: true}

	// render to PNG
	var buf bytes.Buffer
	err := RenderPNG(p.Drawing, &buf)
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
