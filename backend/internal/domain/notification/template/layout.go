package template

import (
	"bytes"
	"encoding/base64"
	"html"
	"html/template"

	_ "embed"
)

//go:embed assets/prova-mark.png
var provaMarkPNG []byte

// logoDataURI is the embedded Prova mark as a data URI. Emails cannot rely on
// a public CDN (no DNS for assets), so the mark ships inline.
//
// Typed as template.URL so html/template does not scrub the data: scheme to
// #ZgotmplZ — the bytes are compile-time embedded, not user input.
var logoDataURI = template.URL("data:image/png;base64," + base64.StdEncoding.EncodeToString(provaMarkPNG))

// Brand palette — mirrors frontend/shared/styles/tokens.css (mist / steel /
// slate / ember). Emails cannot use CSS variables, so the hexes are inlined.
const (
	colorPaper   = "#EAEFEF"
	colorCard    = "#FFFFFF"
	colorLine    = "#BFC9D1"
	colorInk     = "#25343F"
	colorEmber   = "#FF9B51"
	colorMuted   = "#6B7A85"
	fontHeading  = `Georgia, ui-serif, serif`
	fontBody     = `-apple-system, Segoe UI, Roboto, Helvetica, Arial, sans-serif`
	fontMono     = `ui-monospace, SFMono-Regular, Menlo, Consolas, monospace`
	brandTagline = "YETKİNLİK SERTİFİKASYONU"
)

// shellData drives the shared HTML card used by every transactional mail.
type shellData struct {
	LogoDataURI template.URL
	Title       string
	Body        template.HTML
	Footer      string
	MagicLink   string
}

var shellHTML = template.Must(template.New("shell").Parse(`<!doctype html>
<html lang="tr">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>{{.Title}}</title>
</head>
<body style="margin:0;padding:0;background:` + colorPaper + `;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%" style="background:` + colorPaper + `;">
<tr><td align="center" style="padding:32px 16px;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0" width="100%" style="max-width:480px;background:` + colorCard + `;border:1px solid ` + colorLine + `;border-radius:12px;border-top:3px solid ` + colorEmber + `;">
<tr><td style="padding:28px 28px 12px 28px;">
<table role="presentation" cellpadding="0" cellspacing="0" border="0"><tr>
<td style="vertical-align:middle;"><img src="{{.LogoDataURI}}" width="40" height="40" alt="Prova" style="display:block;border:0;outline:none;text-decoration:none;" /></td>
<td style="vertical-align:middle;padding-left:12px;">
<div style="font-family:` + fontHeading + `;font-size:22px;line-height:1.2;color:` + colorInk + `;font-weight:700;">Prova</div>
<div style="font-family:` + fontBody + `;font-size:10px;line-height:1.4;letter-spacing:0.14em;text-transform:uppercase;color:` + colorMuted + `;margin-top:2px;">` + brandTagline + `</div>
</td>
</tr></table>
</td></tr>
<tr><td style="padding:8px 28px 28px 28px;">
<h1 style="margin:0 0 16px;font-family:` + fontHeading + `;font-size:20px;line-height:1.35;font-weight:700;color:` + colorInk + `;">{{.Title}}</h1>
{{.Body}}
{{if .MagicLink}}
<p style="margin:20px 0 12px;font-family:` + fontBody + `;font-size:15px;line-height:1.6;color:` + colorInk + `;">Kodu yazmak istemiyorsanız doğrudan bu bağlantıyla giriş yapabilirsiniz:</p>
<p style="margin:0 0 20px;"><a href="{{.MagicLink}}" style="display:inline-block;background:` + colorInk + `;color:` + colorCard + `;font-family:` + fontBody + `;font-size:15px;font-weight:600;line-height:1;text-decoration:none;padding:12px 20px;border-radius:8px;">Bağlantıyla giriş yap</a></p>
{{end}}
<p style="margin:24px 0 0;font-family:` + fontBody + `;font-size:13px;line-height:1.55;color:` + colorMuted + `;">{{.Footer}}</p>
</td></tr>
</table>
</td></tr>
</table>
</body>
</html>`))

// renderShell wraps trusted inner HTML in the branded card shell.
func renderShell(title, footer string, body template.HTML, magicLink string) string {
	return render(shellHTML, shellData{
		LogoDataURI: logoDataURI,
		Title:       title,
		Body:        body,
		Footer:      footer,
		MagicLink:   magicLink,
	})
}

// otpPanel is the soft code block used by login and e-mail verification.
func otpPanel(code string) template.HTML {
	return template.HTML(
		`<div style="margin:0 0 20px;padding:20px 16px;background:` + colorPaper + `;border-radius:8px;text-align:center;">` +
			`<p style="margin:0;font-family:` + fontMono + `;font-size:34px;font-weight:700;letter-spacing:8px;line-height:1.2;color:` + colorInk + `;">` +
			html.EscapeString(code) +
			`</p></div>`,
	)
}

// bodyParagraph renders a single body paragraph (escaped).
func bodyParagraph(text string) template.HTML {
	return template.HTML(
		`<p style="margin:0 0 16px;font-family:` + fontBody + `;font-size:15px;line-height:1.6;color:` + colorInk + `;">` +
			html.EscapeString(text) +
			`</p>`,
	)
}

// bodyHTML joins trusted fragments into one body block.
func bodyHTML(parts ...template.HTML) template.HTML {
	var b bytes.Buffer
	for _, p := range parts {
		b.WriteString(string(p))
	}
	return template.HTML(b.String())
}

// render executes a parsed template. The templates are compile-time constants
// with escaped data, so execution cannot fail for reasons a caller could act
// on; an empty HTML body still leaves the text alternative intact.
func render(tmpl *template.Template, data any) string {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return ""
	}
	return buf.String()
}
