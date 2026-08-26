package cmd

import "github.com/charmbracelet/lipgloss"

var (
	tuiTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F2C14E"))
	tuiMuted    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	tuiSelected = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#245B75"))
	tuiGet      = lipgloss.NewStyle().Foreground(lipgloss.Color("#61D095")).Bold(true)
	tuiPost     = lipgloss.NewStyle().Foreground(lipgloss.Color("#F2C14E")).Bold(true)
	tuiDelete   = lipgloss.NewStyle().Foreground(lipgloss.Color("#E76F51")).Bold(true)
	tuiBorder   = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B4252"))
	tuiActive   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F2C14E"))
	tuiError    = lipgloss.NewStyle().Foreground(lipgloss.Color("#E76F51"))
	tuiPrompt   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#88C0D0"))
	tuiHeader   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#2E3440"))
	tuiCommand  = lipgloss.NewStyle().Border(lipgloss.NormalBorder(), true, true, true, true).BorderForeground(lipgloss.Color("#88C0D0")).BorderTopForeground(lipgloss.Color("#8FBCBB")).Padding(0, 1)
	tuiPanel    = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#3B4252")).Padding(0, 1)
)
