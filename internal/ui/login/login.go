// Package login renders pairing: QR canvas (modules drawn locally),
// phone-number link toggle with 8-char code, status line (WL-2.1).
package login

import (
	"strconv"

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

// loginSteps mirror the reference client's linking instructions.
var loginSteps = []string{
	"Open WhatsApp on your phone",
	"On Android tap Menu ⋮, on iPhone tap Settings",
	"Tap Linked devices, then Link a device",
	"Scan the QR code to confirm",
}

// New builds the login view: a card with numbered steps beside the
// QR code, and a link that swaps in phone-number pairing. onPhoneCode
// requests an 8-char code for the given number; cancel aborts pairing.
func New(onPhoneCode func(number string) string, cancel func()) *View {
	v := &View{}
	v.Box = gtk.NewBox(gtk.OrientationVertical, 0)
	v.Box.AddCSSClass("login-page")

	card := gtk.NewBox(gtk.OrientationVertical, 20)
	card.AddCSSClass("login-card")
	card.SetHAlign(gtk.AlignCenter)
	card.SetVAlign(gtk.AlignCenter)
	card.SetVExpand(true)
	v.Box.Append(card)

	body := gtk.NewBox(gtk.OrientationHorizontal, 48)
	card.Append(body)

	steps := gtk.NewBox(gtk.OrientationVertical, 18)
	steps.SetVAlign(gtk.AlignCenter)
	title := gtk.NewLabel(i18n.T("login.title"))
	title.AddCSSClass("login-title")
	title.SetXAlign(0)
	steps.Append(title)
	for i, text := range loginSteps {
		row := gtk.NewBox(gtk.OrientationHorizontal, 14)
		num := gtk.NewLabel(strconv.Itoa(i + 1))
		num.AddCSSClass("step-num")
		num.SetVAlign(gtk.AlignStart)
		row.Append(num)
		l := gtk.NewLabel(text)
		l.AddCSSClass("login-step")
		l.SetXAlign(0)
		l.SetWrap(true)
		l.SetMaxWidthChars(34)
		row.Append(l)
		steps.Append(row)
	}
	body.Append(steps)

	frame := gtk.NewBox(gtk.OrientationVertical, 0)
	frame.AddCSSClass("qr-frame")
	frame.SetVAlign(gtk.AlignCenter)
	v.area = gtk.NewDrawingArea()
	v.area.SetContentWidth(264)
	v.area.SetContentHeight(264)
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
		cr.SetSourceRGB(0x0A/255.0, 0x0A/255.0, 0x0A/255.0)
		for y, row := range v.modules {
			for x, black := range row {
				if black {
					cr.Rectangle(float64(offsetX+x*cell), float64(offsetY+y*cell), float64(cell), float64(cell))
					cr.Fill()
				}
			}
		}
	})
	frame.Append(v.area)
	body.Append(frame)

	// Phone-number pairing, hidden behind a link like the reference.
	phone := gtk.NewBox(gtk.OrientationHorizontal, 8)
	phone.SetHAlign(gtk.AlignCenter)
	entry := gtk.NewEntry()
	entry.SetPlaceholderText("Phone number with country code")
	entry.SetWidthChars(26)
	phone.Append(entry)
	linkBtn := gtk.NewButtonWithLabel(i18n.T("login.code"))
	linkBtn.AddCSSClass("suggested-action")
	linkBtn.ConnectClicked(func() {
		code := onPhoneCode(entry.Buffer().Text())
		v.codeView.SetText("Enter on your phone: " + code)
	})
	phone.Append(linkBtn)
	phoneRev := gtk.NewRevealer()
	phoneRev.SetChild(phone)
	card.Append(phoneRev)

	v.codeView = gtk.NewLabel("")
	v.codeView.AddCSSClass("monospace")
	v.codeView.AddCSSClass("login-code")
	card.Append(v.codeView)

	foot := gtk.NewBox(gtk.OrientationHorizontal, 16)
	foot.SetHAlign(gtk.AlignCenter)
	toggle := gtk.NewButtonWithLabel("Log in with phone number")
	toggle.AddCSSClass("link-btn")
	toggle.ConnectClicked(func() {
		show := !phoneRev.RevealChild()
		phoneRev.SetRevealChild(show)
		if show {
			toggle.SetLabel("Log in with QR code")
			entry.GrabFocus()
		} else {
			toggle.SetLabel("Log in with phone number")
		}
	})
	foot.Append(toggle)
	cancelBtn := gtk.NewButtonWithLabel(i18n.T("login.cancel"))
	cancelBtn.AddCSSClass("link-btn")
	cancelBtn.ConnectClicked(cancel)
	foot.Append(cancelBtn)
	card.Append(foot)

	v.status = gtk.NewLabel(i18n.T("login.scan"))
	v.status.AddCSSClass("login-status")
	v.status.SetWrap(true)
	card.Append(v.status)

	lock := gtk.NewBox(gtk.OrientationHorizontal, 6)
	lock.SetHAlign(gtk.AlignCenter)
	lock.SetMarginBottom(28)
	lock.AddCSSClass("login-foot")
	lock.Append(gtk.NewImageFromIconName("wl-lock-symbolic"))
	lock.Append(gtk.NewLabel("Your personal messages are end-to-end encrypted"))
	v.Box.Append(lock)
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
