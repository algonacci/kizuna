package main

import "github.com/egoist/mygo/ui"

var (
	base   = ui.Hex("#101318")
	panel  = ui.Hex("#171b22")
	raised = ui.Hex("#1e242d")
	line   = ui.Hex("#2a323e")
	text   = ui.Hex("#e5eaf1")
	muted  = ui.Hex("#929eaf")
	accent = ui.Hex("#8ac9b2")
	green  = ui.Hex("#94d4ac")
	amber  = ui.Hex("#e9bd7c")
	red    = ui.Hex("#ed9295")
)

func appTheme() *ui.Theme {
	t := *ui.DarkTheme()
	t.Background, t.Surface, t.Border = base, panel, line
	t.Text, t.TextMuted, t.Accent = text, muted, accent
	t.SurfaceHover, t.SurfacePressed = raised, line
	t.FontSize, t.Radius = 13, 7
	return &t
}
func icon(body string) *ui.SVG {
	return ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">` + body + `</svg>`))
}

var (
	icBranch   = icon(`<circle cx="6" cy="5" r="2"/><circle cx="6" cy="19" r="2"/><circle cx="18" cy="5" r="2"/><path d="M6 7v10M18 7c0 7-12 3-12 10"/>`)
	icFolder   = icon(`<path d="M3 7V4h6l2 3h10v13H3Z"/>`)
	icTerminal = icon(`<path d="m4 6 6 6-6 6m9 0h7"/>`)
	icPlus     = icon(`<path d="M12 5v14M5 12h14"/>`)
)

func badge(c *ui.Context, label string, color ui.Color) {
	ui.Text(c, label).FontSize(11).TextColor(color).Padding(4, 8).Radius(4).Background(raised).Shrink(0)
}
func caption(c *ui.Context, label string) { ui.Text(c, label).FontSize(11).TextColor(muted) }
