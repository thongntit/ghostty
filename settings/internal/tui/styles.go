package tui

import "charm.land/lipgloss/v2"

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f5c2e7"))
	accentStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#89dceb"))
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#a6adc8"))
	selectedStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#cba6f7"))
	errorStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#f38ba8"))
	panelStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2)

	appearanceKickerStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f38ba8"))
	appearanceHeroStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f5c2e7"))
	appearanceCardStyle    = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#45475a")).Padding(1, 2)
	appearanceCardTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#89dceb"))
	appearanceCurrentTag   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#a6e3a1")).Background(lipgloss.Color("#313244")).Padding(0, 1)
	appearanceCandidateTag = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#f9e2af")).Background(lipgloss.Color("#45475a")).Padding(0, 1)
	appearanceSelectedRow  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#cba6f7")).Background(lipgloss.Color("#302b46"))
	appearanceHintStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("#7f849c"))
)
