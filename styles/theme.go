package styles

import "github.com/charmbracelet/lipgloss"

var (
	ColorBg         = lipgloss.AdaptiveColor{Dark: "#1a1a1a", Light: "#f5f5f0"}
	ColorFg         = lipgloss.AdaptiveColor{Dark: "#e8e8e8", Light: "#1a1a1a"}
	ColorMuted      = lipgloss.AdaptiveColor{Dark: "#555555", Light: "#999999"}
	ColorBorder     = lipgloss.AdaptiveColor{Dark: "#333333", Light: "#dddddd"}
	ColorAccent     = lipgloss.Color("#FF5F1F")
	ColorAccentSoft = lipgloss.Color("#FF8C42")
	ColorSuccess    = lipgloss.Color("#3dd68c")
	ColorWarning    = lipgloss.Color("#f5a623")
	ColorError      = lipgloss.Color("#ff5f5f")
	ColorInfo       = lipgloss.Color("#5f9fff")
	ColorDiffAdd    = lipgloss.Color("#1e4d2b")
	ColorDiffAddFg  = lipgloss.Color("#3dd68c")
	ColorDiffDel    = lipgloss.Color("#4d1e1e")
	ColorDiffDelFg  = lipgloss.Color("#ff5f5f")
	ColorCodeBg     = lipgloss.AdaptiveColor{Dark: "#232323", Light: "#efefef"}
	ColorCodeBorder = lipgloss.Color("#FF5F1F")
)

var (
	StyleBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(ColorBorder)

	StyleAccentBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(ColorAccent)

	StyleCode = lipgloss.NewStyle().
			Background(ColorCodeBg).
			BorderLeft(true).
			BorderStyle(lipgloss.ThickBorder()).
			BorderLeftForeground(ColorCodeBorder).
			PaddingLeft(1)

	StyleUserMsg = lipgloss.NewStyle().
			Foreground(ColorFg).
			Bold(true)

	StyleAssistantLabel = lipgloss.NewStyle().
				Foreground(ColorAccent).
				Bold(true)

	StyleMuted   = lipgloss.NewStyle().Foreground(ColorMuted)
	StyleSuccess = lipgloss.NewStyle().Foreground(ColorSuccess)
	StyleError   = lipgloss.NewStyle().Foreground(ColorError)
	StyleWarning = lipgloss.NewStyle().Foreground(ColorWarning)
)
