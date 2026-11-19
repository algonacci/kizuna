package main

import (
	"strings"
	"time"

	"github.com/egoist/mygo/ui"
)

// Palette, after Nimbo: charcoal in dark mode, Finder-like greys in light
// mode, one periwinkle accent. The colour variables are swapped by
// usePalette at the start of each frame, on the UI thread.
var (
	base, panel, surface, canvas, raised, line, softLine ui.Color
	text, muted, faint, accent, accentHover, accentPress ui.Color
	accentText, green, amber, red, addBg, delBg, hunkBg  ui.Color
	backdrop, shadow                                     ui.Color
	isDark                                               bool
)

type palette struct {
	dark                                                 bool
	base, panel, surface, canvas, raised, line, softLine string
	text, muted, faint, accent, accentHover, accentPress string
	accentText, green, amber, red, addBg, delBg, hunkBg  string
	backdrop, shadow                                     ui.Color
}

var darkPalette = palette{
	dark: true,
	base: "#0e1118", panel: "#131720", surface: "#161b26", canvas: "#11151e", raised: "#202738", line: "#252c3b", softLine: "#1d2431",
	text: "#e6eaf2", muted: "#9aa3b7", faint: "#6c778c", accent: "#8da9ff", accentHover: "#a5bcff", accentPress: "#7895e9",
	accentText: "#101c3c", green: "#94d4ac", amber: "#e9bd7c", red: "#ed9295", addBg: "#152a24", delBg: "#301e25", hunkBg: "#1a2133",
	backdrop: ui.RGBA(4, 6, 10, 0.68), shadow: ui.RGBA(0, 0, 0, 0.5),
}

var lightPalette = palette{
	base: "#f3f4f7", panel: "#eceef2", surface: "#ffffff", canvas: "#ffffff", raised: "#e1e5ec", line: "#d3d8e0", softLine: "#e0e3e9",
	text: "#1b1f2a", muted: "#566074", faint: "#747d90", accent: "#3b63e0", accentHover: "#5076e6", accentPress: "#2c52c9",
	accentText: "#ffffff", green: "#1d8a57", amber: "#a86400", red: "#cc2f45", addBg: "#e5f5ec", delBg: "#fbe9ec", hunkBg: "#eaeefb",
	backdrop: ui.RGBA(22, 26, 36, 0.36), shadow: ui.RGBA(20, 24, 34, 0.18),
}

func usePalette(p palette) {
	h := ui.Hex
	isDark = p.dark
	base, panel, surface, canvas, raised, line, softLine = h(p.base), h(p.panel), h(p.surface), h(p.canvas), h(p.raised), h(p.line), h(p.softLine)
	text, muted, faint, accent, accentHover, accentPress = h(p.text), h(p.muted), h(p.faint), h(p.accent), h(p.accentHover), h(p.accentPress)
	accentText, green, amber, red, addBg, delBg, hunkBg = h(p.accentText), h(p.green), h(p.amber), h(p.red), h(p.addBg), h(p.delBg), h(p.hunkBg)
	backdrop, shadow = p.backdrop, p.shadow
}

func init() { usePalette(darkPalette) }

var brandGradient = ui.LinearGradient{From: ui.Hex("#779ef5"), To: ui.Hex("#a28aef"), Angle: 135}

// Theme modes, in the order the title bar button cycles through them.
const (
	themeSystem = "system"
	themeLight  = "light"
	themeDark   = "dark"
)

func nextTheme(mode string) string {
	switch mode {
	case themeLight:
		return themeDark
	case themeDark:
		return themeSystem
	}
	return themeLight
}

func appTheme() *ui.Theme {
	t := *ui.LightTheme()
	if isDark {
		t = *ui.DarkTheme()
	}
	t.Background, t.Surface, t.Border = base, surface, line
	t.Text, t.TextMuted, t.Accent = text, muted, accent
	t.SurfaceHover, t.SurfacePressed = raised, line
	t.AccentHover, t.AccentPressed, t.AccentText = accentHover, accentPress, accentText
	t.Danger, t.Warning, t.Success = red, amber, green
	t.Selection, t.Focus = accent.Alpha(0.25), accent.Alpha(0.7)
	t.Scrollbar = ui.RGBA(255, 255, 255, 0.14)
	if !isDark {
		t.Scrollbar = ui.RGBA(0, 0, 0, 0.18)
	}
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
	icTree     = icon(`<rect x="9" y="3" width="6" height="5" rx="1"/><rect x="2" y="16" width="6" height="5" rx="1"/><rect x="16" y="16" width="6" height="5" rx="1"/><path d="M12 8v4H5v4m7-4h7v4"/>`)
	icCheck    = icon(`<path d="m5 12 4 4L19 6"/>`)
	icChevron  = icon(`<path d="m8 10 4 4 4-4"/>`)
	icRepo     = icon(`<path d="M5 19V5a2 2 0 0 1 2-2h12v15H7a2 2 0 0 0-2 2Zm0 0a2 2 0 0 0 2 2h12M9 7h6"/>`)
	icLock     = icon(`<rect x="5" y="11" width="14" height="10" rx="2"/><path d="M8 11V8a4 4 0 0 1 8 0v3"/>`)
	icSun      = icon(`<circle cx="12" cy="12" r="4"/><path d="M12 2v2M12 20v2m-7.07-17.07 1.41 1.41m11.32 11.32 1.41 1.41M2 12h2m16 0h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41"/>`)
	icMoon     = icon(`<path d="M12 3a6 6 0 0 0 9 9 9 9 0 1 1-9-9Z"/>`)
	icMonitor  = icon(`<rect width="20" height="14" x="2" y="3" rx="2"/><path d="M8 21h8M12 17v4"/>`)
	icRemote   = icon(`<circle cx="12" cy="12" r="9"/><path d="M3 12h18M12 3a14 14 0 0 1 0 18M12 3a14 14 0 0 0 0 18"/>`)
)

func caption(c *ui.Context, label string) {
	ui.Text(c, label).FontSize(10).FontWeight(600).LetterSpacing(0.7).TextColor(faint)
}

var hoverFade = ui.ElementTransition{Colors: true, Duration: 120 * time.Millisecond}

// barButton is a GitHub Desktop style toolbar section: an icon, a small
// caption over a bold value, and a chevron when it opens a dropdown.
func barButton(c *ui.Context, ic *ui.SVG, captionText, title, label string, width float32, open, enabled, dropdown bool) ui.Element {
	b := ui.ButtonBase(c).Width(width).Shrink(0).FillHeight().Padding(0, 14).Gap(10).Justify(ui.Start).
		Label(label).Tooltip(title).Disabled(!enabled).TextColor(text).Transition(hoverFade).BorderWidth(0, 1, 0, 0).BorderColor(softLine)
	if !enabled {
		b.Opacity(0.45)
	} else if open {
		b.Background(raised)
	} else if b.Hovered() {
		b.Background(softLine)
	}
	b.Children(func() {
		ui.Icon(c, ic).FontSize(16).TextColor(muted).Shrink(0)
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(2).Children(func() {
			ui.Text(c, captionText).FontSize(10.5).TextColor(muted).SingleLine().FillWidth()
			ui.Text(c, title).FontSize(13).FontWeight(600).SingleLine().FillWidth()
		})
		if dropdown {
			ui.Icon(c, icChevron).FontSize(14).TextColor(muted).Shrink(0)
		}
	})
	return b
}

// popoverPanel styles a dropdown panel the same way everywhere.
func popoverPanel(p ui.Element, width float32) {
	p.Width(width).Padding(6).Gap(2).Background(surface).Border(1, line).Radius(10).Shadow(0, 10, 28, 0, shadow)
}

// menuRow is one clickable row of a dropdown: an icon (or a check for the
// current choice), a title and an optional muted detail line.
func menuRow(c *ui.Context, ic *ui.SVG, title, detail, shortcut, label string, current, enabled bool, fn func()) {
	b := ui.ButtonBase(c).FillWidth().MinHeight(34).Padding(6, 10).Gap(10).Radius(6).Justify(ui.Start).Label(label).Tooltip(detail).Disabled(!enabled).TextColor(text)
	if !enabled {
		b.Opacity(0.45)
	} else if b.Hovered() {
		b.Background(raised)
	}
	b.OnClick(fn).Children(func() {
		color := muted
		if current {
			ic, color = icCheck, accent
		}
		ui.Box(c).Width(16).Shrink(0).Center().Children(func() {
			if ic != nil {
				ui.Icon(c, ic).FontSize(14).TextColor(color)
			}
		})
		ui.Column(c).Grow(1).Basis(0).MinWidth(0).Gap(2).Children(func() {
			t := ui.Text(c, title).FontSize(12.5).SingleLine().FillWidth()
			if current {
				t.FontWeight(600)
			}
			if detail != "" {
				ui.Text(c, detail).FontSize(10.5).TextColor(faint).SingleLine().FillWidth()
			}
		})
		if shortcut != "" {
			ui.Text(c, shortcut).FontSize(11).TextColor(faint).Shrink(0)
		}
	})
}
func menuSeparator(c *ui.Context) {
	ui.Box(c).FillWidth().Height(1).Background(softLine).Margin(4, 0)
}
func menuCaption(c *ui.Context, label string) {
	ui.Text(c, label).FontSize(10).FontWeight(600).LetterSpacing(0.7).TextColor(faint).Padding(8, 10, 4, 10)
}

// dialog is the one modal look of the app: a dimmed backdrop that hides
// what is behind it and a raised panel. Clicking the backdrop or Esc
// clears *open (DialogBase), unless keep says the dialog must stay.
func dialog(c *ui.Context, open *bool, width float32, keep bool, fn func()) {
	shown := *open
	ui.DialogBase(c, &shown, func(bd, p ui.Element) {
		bd.Background(backdrop)
		w, _ := c.Size()
		p.Width(min(width, float32(w)-48)).Padding(22).Gap(14).Background(surface).Radius(12).Border(1, line).Shadow(0, 18, 48, 0, shadow)
		fn()
	})
	if !shown && !keep {
		*open = false
	}
}
func dialogTitle(c *ui.Context, title, subtitle string) {
	ui.Column(c).FillWidth().Gap(5).Children(func() {
		ui.Text(c, title).FontSize(16).FontWeight(700)
		if subtitle != "" {
			ui.Text(c, subtitle).FontSize(12).TextColor(muted).Wrap()
		}
	})
}
func dialogButtons(c *ui.Context, fn func()) {
	ui.Row(c).FillWidth().Gap(8).Padding(6, 0, 0, 0).Children(func() {
		ui.Box(c).Grow(1)
		fn()
	})
}

// statusMark is GitHub Desktop's small square file-status symbol.
func statusMark(c *ui.Context, code string) {
	symbol, color, name := "•", amber, "Modified"
	switch t := strings.TrimSpace(code); {
	case t == "??":
		symbol, color, name = "+", green, "New file"
	case strings.Contains(t, "U") || t == "AA" || t == "DD":
		symbol, color, name = "!", red, "Conflicted"
	case strings.Contains(t, "D"):
		symbol, color, name = "−", red, "Deleted"
	case strings.Contains(t, "R"):
		symbol, color, name = "→", accent, "Renamed"
	case strings.Contains(t, "A"):
		symbol, color, name = "+", green, "Added"
	}
	ui.Box(c).Size(15, 15).Shrink(0).Radius(4).Border(1, color.Alpha(0.75)).Center().Tooltip(name).Children(func() {
		ui.Text(c, symbol).FontSize(11.5).FontWeight(700).TextColor(color)
	})
}
