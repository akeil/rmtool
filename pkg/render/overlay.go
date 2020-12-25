package render

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/jung-kurt/gofpdf"
	"github.com/phpdave11/gofpdi"
    "github.com/pdfcpu/pdfcpu/pkg/api"
    "github.com/pdfcpu/pdfcpu/pkg/pdfcpu"

	"akeil.net/akeil/rm"
	"akeil.net/akeil/rm/internal/logging"
)
func overlayPDF(doc *rm.Document, w io.Writer) error {
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

    // set up drawings
    stamps := make(map[int]*pdfcpu.Watermark)
    for i, pageId := range doc.Pages() {
        d, err := doc.Drawing(pageId)
		if err != nil {
			logging.Info("Page %d has no drawing", i)
            continue
		}


    }

    return api.AddWatermarksMap(rs, w, drawings, config)
}


func _overlayPDF(doc *rm.Document, pdf gofpdf.Pdf) error {
	logging.Debug("Render PDF with overlay")

	// we need to read the underlaying PDF doc
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
	logging.Debug("Read original PDF")
	err = dontPanic(func() {
		im.SetSourceStream(&rs)
	})
    if err != nil {
		return err
	}
    fmt.Println(im)

	logging.Debug("Did read original PDF")
	logging.Debug("Original PDF has %v pages", im.GetNumPages())
	logging.Debug("Page sizes are %v", im.GetPageSizes())
    sizes := im.GetPageSizes()

	for i, pageId := range doc.Pages() {
		pdf.AddPage()

		p, err := doc.Page(pageId)
		if err != nil {
			return err
		}
		logging.Info("Page has orientation %v", p.Orientation())
        logging.Info("sizes: %v", sizes[int(p.Number())])

        // TODO: how do we know which box to use?
		im.ImportPage(int(p.Number()), "/MediaBox")

		d, err := doc.Drawing(pageId)
		if err != nil {
			logging.Info("Page %d has no drawing", i)
		}
		if d != nil {
			logging.Debug("overlay the drawing")
		}
		// import the underlay page from src
		// paint on paint
		// add OPTIONAL drawing as overlay
	}

	return nil
}

// executes the given function in a separate go routine.
// If that panics, this will recover and return the panic as an error.
func dontPanic(f func()) error {
	rv := make(chan error, 0)

	go func() {
		defer func() {
			x := recover()
			if x != nil {
				logging.Debug("did recover: %v", x)
				rv <- fmt.Errorf("recovered from: %v", x)
			}
			rv <- nil
		}()

		fmt.Println("run...")
		f()
		fmt.Println("exiting...")
	}()
	fmt.Println("wait for it")
	return <-rv
}

/*


func readPDF(src string) error {
    tpl := gofpdi.ImportPage(pdf, src, 1, "/MediaBox")
    gofpdi.UseImportedTemplate(pdf, tpl, 20, 50, 150, 0)

    tpl := gofpdi.ImportPageFromStream(pdf, reader, 1, "/TrimBox")

    gofpdi.UseImportedTemplate(pdf, tpl, 20, 50, 150, 0)

    return nil
}
*/
