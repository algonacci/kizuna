package main

import (
	"github.com/egoist/mygo/ui"
	"time"
)

var (
	base     = ui.Hex("#0e1118")
	panel    = ui.Hex("#131720")
	raised   = ui.Hex("#202738")
	line     = ui.Hex("#252c3b")
	text     = ui.Hex("#e6eaf2")
	muted    = ui.Hex("#929bb0")
	accent   = ui.Hex("#8da9ff")
	faint    = ui.Hex("#626e84")
	softLine = ui.Hex("#1d2431")
	canvas   = ui.Hex("#11151e")
	green    = ui.Hex("#94d4ac")
	amber    = ui.Hex("#e9bd7c")
	red      = ui.Hex("#ed9295")
)

func appTheme() *ui.Theme {
	t := *ui.DarkTheme()
	t.Background, t.Surface, t.Border = base, panel, line
	t.Text, t.TextMuted, t.Accent = text, muted, accent
	t.SurfaceHover, t.SurfacePressed = raised, line
	t.AccentHover, t.AccentPressed, t.AccentText = ui.Hex("#a5bcff"), ui.Hex("#7895e9"), ui.Hex("#101c3c")
	t.Selection, t.Focus = accent.Alpha(0.22), accent.Alpha(0.7)
	t.Scrollbar = ui.RGBA(255, 255, 255, 0.14)
	t.FontSize, t.Radius = 13, 8
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
	icClone    = icon(`<rect x="8" y="8" width="13" height="13" rx="2"/><path d="M16 8V3H3v13h5"/>`)
	icRefresh  = icon(`<path d="M20 7v5h-5M4 17v-5h5"/><path d="M6 6a8 8 0 0 1 14 6M18 18a8 8 0 0 1-14-6"/>`)
	icFetch    = icon(`<path d="M5 17a5 5 0 0 1-1-10 7 7 0 0 1 13-1 5 5 0 0 1 2 10M12 11v10m-4-4 4 4 4-4"/>`)
	icPull     = icon(`<path d="M12 3v12m-5-5 5 5 5-5M4 17v4h16v-4"/>`)
	icPush     = icon(`<path d="M12 15V3m-5 5 5-5 5 5M4 17v4h16v-4"/>`)
	icChanges  = icon(`<path d="M14 2H6v20h14V8Z M14 2v6h6M9 12h6M9 16h6"/>`)
	icTree     = icon(`<rect x="9" y="3" width="6" height="5" rx="1"/><rect x="2" y="16" width="6" height="5" rx="1"/><rect x="16" y="16" width="6" height="5" rx="1"/><path d="M12 8v4H5v4m7-4h7v4"/>`)
	icCheck    = icon(`<path d="m5 12 4 4L19 6"/>`)
	icChevron  = icon(`<path d="m8 10 4 4 4-4"/>`)
)

func badge(c *ui.Context, label string, color ui.Color) {
	ui.Text(c, label).FontSize(11).TextColor(color).Padding(4, 8).Radius(4).Background(raised).Shrink(0)
}
func caption(c *ui.Context, label string) {
	ui.Text(c, label).FontSize(10).FontWeight(600).LetterSpacing(0.7).TextColor(faint)
}

var hoverFade = ui.ElementTransition{Colors: true, Duration: 120 * time.Millisecond}

// Native icon-over-label tools, shared with the visual language of Nimbo.
func tool(c *ui.Context, ic *ui.SVG, label, identity string, enabled, on bool, fn func()) {
	width, _ := c.Size()
	size := float32(66)
	if width < 1000 {
		size = 54
	}
	b := ui.ButtonBase(c).Width(size).Height(52).Column().Center().Gap(5).Radius(8).
		Label(identity).Tooltip(identity).Disabled(!enabled).TextColor(muted).Transition(hoverFade)
	if !enabled {
		b.Opacity(0.4)
	} else if on {
		b.Background(accent.Alpha(0.13)).TextColor(accent)
	} else if b.Hovered() {
		b.Background(raised).TextColor(text)
	}
	b.OnClick(fn).Children(func() { ui.Icon(c, ic).FontSize(18); ui.Text(c, label).FontSize(10.5).FontWeight(500) })
}
func toolSeparator(c *ui.Context) { ui.Box(c).Size(1, 28).Background(softLine).Margin(0, 6).Shrink(0) }

// Compact trigger; full disambiguating paths remain available in its popup.
func picker(c *ui.Context, selected *string, options []string, label, display string, enabled bool, fn func()) {
	sel := ui.SelectBase(c, selected)
	sel.Trigger.FillWidth().Height(32).Padding(0, 10).Gap(8).Radius(6).Background(raised).Label(label).Disabled(!enabled).OnChange(fn).Children(func() {
		ui.Text(c, display).FontSize(12).FontWeight(550).SingleLine().Grow(1).MinWidth(0)
		ui.Icon(c, icChevron).FontSize(13).TextColor(muted).Shrink(0)
	})
	sel.Popup(func(p ui.Element) {
		p.Padding(5).Background(panel).Border(1, line).Radius(8)
		for _, option := range options {
			item := sel.Item(option).Padding(9, 12).Radius(5).Label(option)
			if item.Highlighted() {
				item.Background(raised)
			}
			item.Children(func() { ui.Text(c, option).FontSize(12).SingleLine().MaxWidth(560) })
		}
	})
}
