package render

import (
	"bytes"
	"fmt"
	"io"

	"akeil.net/akeil/rm"
	"github.com/jung-kurt/gofpdf"
)

func RenderPDF(n *rm.Notebook, w io.Writer) error {
	pdf := setupPDF()

	for i, p := range n.Pages {
		err := doRenderPDFPage(pdf, p, i)
		if err != nil {
			return err
		}
	}

	return pdf.Output(w)
}

func RenderPDFPage(p *rm.Page, w io.Writer) error {
	pdf := setupPDF()

	err := doRenderPDFPage(pdf, p, 0)
	if err != nil {
		return err
	}

	return pdf.Output(w)
}

func setupPDF() *gofpdf.Fpdf {
	orientation := "P" // [P]ortrait or [L]andscape
	sizeUnit := "pt"
	pageSize := "A4" // or Letter
	fontDir := ""

	pdf := gofpdf.New(orientation, sizeUnit, pageSize, fontDir)

	pdf.SetTopMargin(24)
	pdf.AliasNbPages("{totalPages}")
	pdf.SetFont("helvetica", "", 10)

	pdf.SetFooterFunc(func() {
		pdf.Cellf(0, 10, "%d / {totalPages}", pdf.PageNo())
	})

	return pdf
}

func doRenderPDFPage(pdf *gofpdf.Fpdf, p *rm.Page, i int) error {
	// TODO: determine orientation, rotate image if neccessary
	// and set the page to Landscape
	pdf.AddPage()

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
