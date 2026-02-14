package ui

import "github.com/charmbracelet/lipgloss"

// mine's color palette — warm golds, cool stone, bright gems.
var (
	// Primary colors
	Gold     = lipgloss.Color("#FFD700")
	Amber    = lipgloss.Color("#FFBF00")
	Copper   = lipgloss.Color("#B87333")
	Stone    = lipgloss.Color("#8B8680")
	Deep     = lipgloss.Color("#2D2D2D")
	Emerald  = lipgloss.Color("#50C878")
	Ruby     = lipgloss.Color("#E0115F")
	Sapphire = lipgloss.Color("#0F52BA")
	Dim      = lipgloss.Color("#666666")
	Bright   = lipgloss.Color("#FFFFFF")
	Subtle   = lipgloss.Color("#AAAAAA")

	// Semantic styles
	Title = lipgloss.NewStyle().
		Bold(true).
		Foreground(Gold)

	Subtitle = lipgloss.NewStyle().
			Foreground(Amber)

	Success = lipgloss.NewStyle().
		Foreground(Emerald)

	Error = lipgloss.NewStyle().
		Foreground(Ruby)

	Warning = lipgloss.NewStyle().
		Foreground(Amber)

	Info = lipgloss.NewStyle().
		Foreground(Sapphire)

	Muted = lipgloss.NewStyle().
		Foreground(Dim)

	Accent = lipgloss.NewStyle().
		Foreground(Gold).
		Bold(true)

	// Component styles
	Banner = lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(Gold).
		Padding(0, 1)

	Tag = lipgloss.NewStyle().
		Foreground(Bright).
		Background(Copper).
		Padding(0, 1).
		Bold(true)

	KeyStyle = lipgloss.NewStyle().
			Foreground(Amber).
			Bold(true)

	ValueStyle = lipgloss.NewStyle().
			Foreground(Bright)
)

// Icon constants — consistent emoji language.
const (
	IconPick    = "⛏ "
	IconGem     = "💎"
	IconGold    = "🪙"
	IconTodo    = "📋"
	IconDone    = "✅"
	IconOverdue = "🔴"
	IconTools   = "🔧"
	IconPackage = "📦"
	IconVault   = "🔑"
	IconGrow    = "🌱"
	IconStar    = "⭐"
	IconFire    = "🔥"
	IconWarn    = "⚠️ "
	IconError   = "✗ "
	IconOk      = "✓ "
	IconArrow   = "→"
	IconDot     = "·"
	IconDig     = "⛏ "
)
