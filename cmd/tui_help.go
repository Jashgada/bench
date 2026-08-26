package cmd

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type helpSection struct {
	title string
	rows  [][2]string
}

func helpSections() []helpSection {
	return []helpSection{
		{"General", [][2]string{
			{"j / k, arrows", "move cursor / scroll response"},
			{"enter", "open operation / execute form"},
			{"esc", "back / clear filter"},
			{"q, ctrl+c", "quit"},
			{"?", "toggle this help"},
		}},
		{"List", [][2]string{
			{"/", "filter operations"},
			{"N / M / T", "sort by name / method / tag (toggle direction)"},
			{":", "command mode"},
		}},
		{"Detail", [][2]string{
			{"tab, left/right", "switch response body / headers"},
			{"r", "run request (re-runs history entry)"},
			{"c", "copy response body"},
			{"y", "copy as curl command"},
			{"/", "search response body"},
			{"enter / n / N", "next / previous search match"},
			{"[ / ]", "older / newer request history"},
		}},
		{"Commands", [][2]string{
			{":list", "back to operation list"},
			{":projects, :p", "project picker"},
			{":project <name>", "switch project"},
			{":group <tag>", "filter by tag (:group all clears)"},
			{":reload", "reload project from disk"},
			{":update", "refresh project from source spec"},
			{":help", "open help overlay"},
			{":q, :quit", "quit bench"},
		}},
	}
}

func (m tuiModel) helpView() string {
	var b strings.Builder
	b.WriteString(tuiTitle.Render("bench keybindings") + "\n\n")
	width := len("command")
	for _, section := range helpSections() {
		for _, row := range section.rows {
			if l := len(row[0]); l > width {
				width = l
			}
		}
	}
	for i, section := range helpSections() {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(tuiActive.Render(section.title) + "\n")
		for _, row := range section.rows {
			b.WriteString(fmt.Sprintf("  %-*s  %s\n", width, row[0], tuiMuted.Render(row[1])))
		}
	}
	b.WriteString("\n" + tuiMuted.Render("press any key to close"))
	return tuiPanel.Render(b.String())
}

func helpOverlayView(model tuiModel) string {
	panel := model.helpView()
	if model.width > 4 {
		panel = lipgloss.NewStyle().MaxWidth(model.width - 4).Render(panel)
	}
	return lipgloss.Place(model.width, model.height, lipgloss.Center, lipgloss.Center, panel)
}
