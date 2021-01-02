package render

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/jung-kurt/gofpdf"
	"github.com/jung-kurt/gofpdf/contrib/gofpdi"

	"github.com/akeil/rm"
	"github.com/akeil/rm/internal/errors"
	"github.com/akeil/rm/internal/logging"
)

func overlayPdf(c *Context, doc *rm.Document, pdf *gofpdf.Fpdf) error {
	logging.Debug("Render PDF with overlay")

	// Read the underlaying PDF doc
	r, err := doc.AttachmentReader()
	if err != nil {
		return err
	}
	defer r.Close()

	// We need a ReadSeeker, so we load the full PDF into memory
	// and create one from the buffer.
	data, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}
	rs := io.ReadSeeker(bytes.NewReader(data))

	im := gofpdi.NewImporter()

	for i, pageID := range doc.Pages() {
		pdf.AddPage()

		var tplID int
		err = dontPanic(func() {
			// TODO: how do we know which box to use?
			tplID = im.ImportPageFromStream(pdf, &rs, i+1, "/MediaBox")
		})
		if err != nil {
			return err
		}
		// Setting h, w to 0 fills the page
		im.UseImportedTemplate(pdf, tplID, 0, 0, 0, 0)

		// Paint the drawing over the original
		d, err := doc.Drawing(pageID)
		if errors.IsNotFound(err) {
			// Not every page has a drawing
			logging.Info("Skip page %d without drawing", i)
			continue
		} else if err != nil {
			return err
		}

		logging.Debug("overlay the drawing for page %v", i)
		err = drawingToPdf(c, pdf, d)
		if err != nil {
			return err
		}
	}

	return nil
}

// dontPanic executes the given function in a separate goroutine.
// If that panics, it will recover and return the panic as an error.
func dontPanic(f func()) error {
	rv := make(chan error)

	go func() {
		// this will "catch" any panic and send its mssage to the error channel
		defer func() {
			x := recover()
			if x != nil {
				logging.Warning("Panic occured (revoered): %v", x)
				rv <- fmt.Errorf("recovered from: %v", x)
			}
			rv <- nil
		}()

		// the actual call that might panic
		f()
	}()

	// wait for the result
	return <-rv
}
