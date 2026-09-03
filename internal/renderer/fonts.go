package renderer

import (
	_ "embed"

	"github.com/jung-kurt/gofpdf"
)

//go:embed fonts/CourierPrime-Regular.ttf
var courierPrimeRegular []byte

//go:embed fonts/CourierPrime-Bold.ttf
var courierPrimeBold []byte

//go:embed fonts/CourierPrime-Italic.ttf
var courierPrimeItalic []byte

//go:embed fonts/CourierPrime-BoldItalic.ttf
var courierPrimeBoldItalic []byte

const fontFamily = "CourierPrime"

func registerCourierPrime(pdf *gofpdf.Fpdf) {
	pdf.AddUTF8FontFromBytes(fontFamily, "", courierPrimeRegular)
	pdf.AddUTF8FontFromBytes(fontFamily, "B", courierPrimeBold)
	pdf.AddUTF8FontFromBytes(fontFamily, "I", courierPrimeItalic)
	pdf.AddUTF8FontFromBytes(fontFamily, "BI", courierPrimeBoldItalic)
}
