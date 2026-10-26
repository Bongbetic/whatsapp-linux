// Package login renders pairing: QR canvas (modules drawn locally),
// phone-number link toggle with 8-char code, status line (WL-2.1).
package login

import (
	"github.com/diamondburned/gotk4/pkg/cairo"
	"github.com/diamondburned/gotk4/pkg/gtk/v4"
	"github.com/skip2/go-qrcode"

	"github.com/Bongbetic/whatsapp-linux/internal/i18n"
)

// Bitmap renders the QR modules for code (pure, headless-testable).
func Bitmap(code string) ([][]bool, error) {
	qr, err := qrcode.New(code, qrcode.Medium)
	if err != nil {
		return nil, err
	}
	return qr.Bitmap(), nil
}

// View is the login screen widget.
type View struct {
	Box      *gtk.Box
	area     *gtk.DrawingArea
	status   *gtk.Label
	codeView *gtk.Label
	modules  [][]bool
}

// New builds the login view. onPhoneCode requests an 8-char code for
// the given number; cancel aborts pairing.
func New(onPhoneCode func(number string) string, cancel func()) *View {
	v := &View{}
	v.Box = gtk.NewBox(gtk.OrientationVertical, 12)
	v.Box.SetHAlign(gtk.AlignCenter)
	v.Box.SetVAlign(gtk.AlignCenter)

	title := gtk.NewLabel(i18n.T("login.title"))
	title.AddCSSClass("title-1")
	v.Box.Append(title)

	v.area = gtk.NewDrawingArea()
	v.area.SetContentWidth(280)
	v.area.SetContentHeight(280)
	v.area.SetDrawFunc(func(_ *gtk.DrawingArea, cr *cairo.Context, w, h int) {
		cr.SetSourceRGB(1, 1, 1)
		cr.Rectangle(0, 0, float64(w), float64(h))
		cr.Fill()
		if len(v.modules) == 0 {
			return
		}
		n := len(v.modules)
		offsetX, offsetY, cell := qrLayout(w, h, n)
		cr.SetAntialias(cairo.AntialiasNone)
		cr.SetSourceRGB(0, 0, 0)
		for y, row := range v.modules {
			for x, black := range row {
				if black {
					cr.Rectangle(float64(offsetX+x*cell), float64(offsetY+y*cell), float64(cell), float64(cell))
					cr.Fill()
				}
			}
		}
	})
	v.Box.Append(v.area)

	v.codeView = gtk.NewLabel("")
	v.codeView.AddCSSClass("monospace")
	v.Box.Append(v.codeView)

	row := gtk.NewBox(gtk.OrientationHorizontal, 6)
	entry := gtk.NewEntry()
	entry.SetPlaceholderText("Phone number for link code")
	row.Append(entry)
	linkBtn := gtk.NewButtonWithLabel(i18n.T("login.code"))
	linkBtn.ConnectClicked(func() {
		code := onPhoneCode(entry.Buffer().Text())
		v.codeView.SetText("Enter on your phone: " + code)
	})
	row.Append(linkBtn)
	v.Box.Append(row)

	v.status = gtk.NewLabel(i18n.T("login.scan"))
	v.Box.Append(v.status)

	cancelBtn := gtk.NewButtonWithLabel(i18n.T("login.cancel"))
	cancelBtn.ConnectClicked(cancel)
	v.Box.Append(cancelBtn)
	return v
}

// ShowQR draws a fresh pairing code.
func (v *View) ShowQR(code string) {
	mods, err := Bitmap(code)
	if err != nil {
		v.SetStatus("QR error: " + err.Error())
		return
	}
	v.modules = mods
	v.codeView.SetText("")
	v.area.QueueDraw()
}

// SetStatus updates the status line.
func (v *View) SetStatus(s string) { v.status.SetText(s) }

// qrLayout returns the module size and origin within the drawing area.
func qrLayout(w, h, modules int) (x, y, cell int) {
	if modules <= 0 {
		return 0, 0, 0
	}
	cell = min(w, h) / modules
	return (w - modules*cell) / 2, (h - modules*cell) / 2, cell
}
