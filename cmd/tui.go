package cmd

import (
	"fmt"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	tuiTitle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F2C14E"))
	tuiMuted    = lipgloss.NewStyle().Foreground(lipgloss.Color("#6B7280"))
	tuiSelected = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#245B75"))
	tuiGet      = lipgloss.NewStyle().Foreground(lipgloss.Color("#61D095")).Bold(true)
	tuiPost     = lipgloss.NewStyle().Foreground(lipgloss.Color("#F2C14E")).Bold(true)
	tuiDelete   = lipgloss.NewStyle().Foreground(lipgloss.Color("#E76F51")).Bold(true)
)

type tuiModel struct {
	project  Project
	items    []API
	filtered []API
	cursor   int
	filter   string
	detail   bool
	width    int
	height   int
}

func runTUI(name string) error {
	project, err := loadProject(name)
	if err != nil {
		return err
	}
	items := append([]API(nil), project.APIs...)
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	model := tuiModel{project: project, items: items, filtered: items}
	_, err = tea.NewProgram(model, tea.WithAltScreen()).Run()
	return err
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			if m.detail {
				m.detail = false
			} else {
				m.filter = ""
				m.applyFilter()
			}
		case "up", "k":
			if !m.detail && m.cursor > 0 {
				m.cursor--
			}
		case "down", "j":
			if !m.detail && m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case "enter":
			if len(m.filtered) > 0 {
				m.detail = true
			}
		case "/":
			if !m.detail {
				m.filter = ""
			}
		case "backspace":
			if !m.detail && len(m.filter) > 0 {
				m.filter = m.filter[:len(m.filter)-1]
				m.applyFilter()
			}
		default:
			if !m.detail && len(msg.String()) == 1 && msg.String() >= " " && msg.String() <= "~" {
				m.filter += msg.String()
				m.applyFilter()
			}
		}
	}
	return m, nil
}

func (m *tuiModel) applyFilter() {
	needle := strings.ToLower(m.filter)
	m.filtered = make([]API, 0, len(m.items))
	for _, api := range m.items {
		if needle == "" || strings.Contains(strings.ToLower(api.Name+" "+api.Method+" "+api.Path), needle) {
			m.filtered = append(m.filtered, api)
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
}

func (m tuiModel) View() string {
	if m.detail {
		return m.detailView()
	}
	var b strings.Builder
	b.WriteString(tuiTitle.Render(" bench ") + tuiMuted.Render("/ API browser") + "\n")
	b.WriteString(fmt.Sprintf(" Project: %s", m.project.Name))
	if m.filter != "" {
		b.WriteString("  Filter: " + m.filter)
	}
	b.WriteString("\n\n")
	b.WriteString(tuiMuted.Render("  METHOD   PATH                                      OPERATION") + "\n")
	for i, api := range m.filtered {
		method := methodStyle(api.Method).Render(fmt.Sprintf("%-8s", api.Method))
		line := fmt.Sprintf("  %s %-42s %s", method, api.Path, api.Name)
		if i == m.cursor {
			line = tuiSelected.Render("› " + strings.TrimPrefix(line, "  "))
		}
		b.WriteString(line + "\n")
	}
	b.WriteString("\n" + tuiMuted.Render(" j/k or arrows  move   enter  inspect   /  filter   q  quit"))
	return b.String()
}

func (m tuiModel) detailView() string {
	api := m.filtered[m.cursor]
	var b strings.Builder
	b.WriteString(tuiTitle.Render(" bench ") + tuiMuted.Render("/ operation detail") + "\n\n")
	b.WriteString(methodStyle(api.Method).Render(api.Method) + "  " + api.Name + "\n")
	b.WriteString("\n" + tuiMuted.Render("URL") + "  " + strings.TrimRight(m.project.BaseURL, "/") + api.Path + "\n")
	if len(api.PathParams)+len(api.QueryParams)+len(api.Headers) > 0 {
		b.WriteString("\n" + tuiMuted.Render("PARAMETERS") + "\n")
		for _, p := range append(append(append([]Parameter{}, api.PathParams...), api.QueryParams...), api.Headers...) {
			required := "optional"
			if p.Required {
				required = "required"
			}
			b.WriteString(fmt.Sprintf("  %-20s %s\n", p.Name, tuiMuted.Render(required)))
		}
	}
	if len(api.RequestBodySchema) > 0 {
		b.WriteString("\n" + tuiMuted.Render("REQUEST BODY") + "  JSON schema available\n")
	}
	b.WriteString("\n" + tuiMuted.Render(" esc  back   q  quit"))
	return b.String()
}

func methodStyle(method string) lipgloss.Style {
	switch method {
	case "GET":
		return tuiGet
	case "POST", "PUT", "PATCH":
		return tuiPost
	case "DELETE":
		return tuiDelete
	default:
		return tuiMuted
	}
}
