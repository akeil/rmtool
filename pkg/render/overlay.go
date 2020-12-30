package render

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/jung-kurt/gofpdf"
	"github.com/jung-kurt/gofpdf/contrib/gofpdi"

	"akeil.net/akeil/rm"
	"akeil.net/akeil/rm/internal/logging"
)

func overlayPDF(c *Context, doc *rm.Document, pdf *gofpdf.Fpdf) error {
	logging.Debug("Render PDF with overlay")

	// Read the underlaying PDF doc
	pr, err := doc.AttachmentReader()
	if err != nil {
		return err
	}
	defer pr.Close()

	// we need a ReadSeeker, so we load th e full PDF into memory
	// and create one from the buffer
	data, err := ioutil.ReadAll(pr)
	if err != nil {
		return err
	}
	rs := io.ReadSeeker(bytes.NewReader(data))

	im := gofpdi.NewImporter()

	for i, pageID := range doc.Pages() {
		pdf.AddPage()

		var tpl int
		err = dontPanic(func() {
			// TODO: how do we know which box to use?
			tpl = im.ImportPageFromStream(pdf, &rs, i+1, "/MediaBox")
		})
		if err != nil {
			return err
		}
		// setting h, w to 0 fills the page
		im.UseImportedTemplate(pdf, tpl, 0, 0, 0, 0)

		d, err := doc.Drawing(pageID)
		if rm.IsNotFound(err) {
			logging.Info("Skip page %d without drawing", i)
			continue
		} else if err != nil {
			return err
		}

		logging.Debug("overlay the drawing for page %v", i)
		err = drawingToPDF(c, pdf, d)
		if err != nil {
			return err
		}
	}

	return nil
}

// executes the given function in a separate go routine.
// If that panics, this will recover and return the panic as an error.
func dontPanic(f func()) error {
	rv := make(chan error, 0)

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
