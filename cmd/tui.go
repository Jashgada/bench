package cmd

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"sort"
	"strings"
	"time"

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
	tuiBorder   = lipgloss.NewStyle().Foreground(lipgloss.Color("#3B4252"))
	tuiActive   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#F2C14E"))
	tuiError    = lipgloss.NewStyle().Foreground(lipgloss.Color("#E76F51"))
	tuiPrompt   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#88C0D0"))
	tuiHeader   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#FFFFFF")).Background(lipgloss.Color("#2E3440"))
	tuiCommand  = lipgloss.NewStyle().Border(lipgloss.NormalBorder()).BorderForeground(lipgloss.Color("#88C0D0")).Padding(0, 1)
)

type tuiState int

const (
	stateList tuiState = iota
	stateDetail
	stateForm
)

type cmdMode int

const (
	cmdNormal cmdMode = iota
	cmdSearch
	cmdCommand
)

type formField struct {
	label    string
	value    string
	required bool
	isBody   bool
}

type responseMsg struct {
	result ResponseResult
}

type clearCopyMsg struct{}

type tuiModel struct {
	project  Project
	items    []API
	filtered []API
	cursor   int
	filter   string
	state    tuiState
	width    int
	height   int

	fields     []formField
	fieldIndex int

	response   *ResponseResult
	executing  bool
	copyStatus string

	mode  cmdMode
	input string
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
	case responseMsg:
		m.executing = false
		m.response = &msg.result
		m.state = stateDetail
		return m, nil
	case clearCopyMsg:
		m.copyStatus = ""
		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		if m.mode != cmdNormal {
			return m.updateCommandBar(msg)
		}
		return m.updateNormal(msg)
	}
	return m, nil
}

func (m tuiModel) updateCommandBar(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = cmdNormal
		m.input = ""
	case "enter":
		cmd := m.input
		mode := m.mode
		m.input = ""
		m.mode = cmdNormal
		if mode == cmdSearch {
			m.applyFilter()
			return m, nil
		}
		return m.executeCommand(cmd)
	case "backspace":
		if len(m.input) > 0 {
			m.input = m.input[:len(m.input)-1]
			if m.mode == cmdSearch {
				m.filter = m.input
				m.applyFilter()
			}
		}
	case "ctrl+c":
		return m, tea.Quit
	default:
		s := msg.String()
		for _, r := range s {
			if r >= ' ' && r <= '~' {
				m.input += string(r)
				if m.mode == cmdSearch {
					m.filter = m.input
					m.applyFilter()
				}
			}
		}
	}
	return m, nil
}

func (m tuiModel) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "esc":
		switch m.state {
		case stateForm:
			m.state = stateDetail
		case stateDetail:
			m.state = stateList
			m.response = nil
		case stateList:
			m.filter = ""
			m.applyFilter()
		}
	case "up", "k":
		switch m.state {
		case stateList:
			if m.cursor > 0 {
				m.cursor--
			}
		case stateForm:
			if m.fieldIndex > 0 {
				m.fieldIndex--
			}
		}
	case "down", "j":
		switch m.state {
		case stateList:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
			}
		case stateForm:
			if m.fieldIndex < len(m.fields)-1 {
				m.fieldIndex++
			}
		}
	case "tab":
		if m.state == stateForm && m.fieldIndex < len(m.fields)-1 {
			m.fieldIndex++
		}
	case "enter":
		switch m.state {
		case stateList:
			if len(m.filtered) > 0 {
				m.state = stateDetail
				m.response = nil
			}
		case stateDetail:
			if len(m.filtered) > 0 {
				m.state = stateList
			}
		case stateForm:
			return m, m.submitForm()
		}
	case "r":
		if m.state == stateDetail && len(m.filtered) > 0 && !m.executing {
			m.buildForm()
			m.state = stateForm
		}
	case "c":
		if m.state == stateDetail && m.response != nil {
			m.copyResponse()
			return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearCopyMsg{} })
		}
	case "/":
		m.mode = cmdSearch
		m.input = m.filter
	case ":":
		m.mode = cmdCommand
		m.input = ""
	}
	return m, nil
}

func (m tuiModel) executeCommand(cmd string) (tea.Model, tea.Cmd) {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return m, nil
	}
	parts := strings.Fields(cmd)
	switch parts[0] {
	case "q", "quit", "exit":
		return m, tea.Quit
	case "list":
		m.state = stateList
		m.response = nil
		return m, nil
	case "help", "h":
		return m, nil
	default:
		for _, api := range m.items {
			if api.Name == parts[0] {
				for i, a := range m.filtered {
					if a.Name == api.Name {
						m.cursor = i
						break
					}
				}
				m.state = stateDetail
				m.response = nil
				return m, nil
			}
		}
	}
	return m, nil
}

func (m *tuiModel) buildForm() {
	api := m.filtered[m.cursor]
	m.fields = nil
	m.fieldIndex = 0
	for _, p := range api.PathParams {
		m.fields = append(m.fields, formField{label: p.Name, required: p.Required})
	}
	for _, p := range api.QueryParams {
		m.fields = append(m.fields, formField{label: p.Name, required: p.Required})
	}
	for _, h := range api.Headers {
		m.fields = append(m.fields, formField{label: h.Name, required: h.Required})
	}
	if len(api.RequestBodySchema) > 0 {
		m.fields = append(m.fields, formField{label: "body", isBody: true, required: api.RequestBodyRequired})
	}
}

func (m tuiModel) submitForm() tea.Cmd {
	api := m.filtered[m.cursor]
	params := map[string]string{}
	var body []byte
	for _, f := range m.fields {
		if f.isBody {
			body = []byte(f.value)
		} else {
			params[f.label] = f.value
		}
	}
	m.executing = true
	return func() tea.Msg {
		result := executeRequest(m.project, api, params, body)
		return responseMsg{result: result}
	}
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
	var content string
	switch m.state {
	case stateForm:
		content = m.formView()
	case stateDetail:
		content = m.detailView()
	default:
		content = m.listView()
	}

	if m.response != nil || m.executing {
		content = lipgloss.JoinVertical(lipgloss.Left, content, m.bottomView())
	}

	return lipgloss.JoinVertical(lipgloss.Left, m.commandBar(), content)
}

func (m tuiModel) commandBar() string {
	prompt := ">"
	text := m.input
	switch m.mode {
	case cmdSearch:
		prompt = "/"
		text = m.input
	case cmdCommand:
		prompt = ":"
		text = m.input
	}
	bar := tuiPrompt.Render(prompt) + " " + text
	if m.mode != cmdNormal {
		bar += "█"
	}
	if m.width > 4 {
		return tuiCommand.Width(m.width - 4).Render(bar)
	}
	return tuiCommand.Render(bar)
}

func (m tuiModel) listView() string {
	var b strings.Builder
	b.WriteString(m.headerView("APIs") + "\n")
	b.WriteString(tuiMuted.Render("  METHOD   PATH                                      OPERATION") + "\n")
	for i, api := range m.filtered {
		method := methodStyle(api.Method).Render(fmt.Sprintf("%-8s", api.Method))
		line := fmt.Sprintf("  %s %-42s %s", method, api.Path, api.Name)
		if i == m.cursor {
			line = tuiSelected.Render("› " + strings.TrimPrefix(line, "  "))
		}
		b.WriteString(line + "\n")
	}
	return b.String()
}

func (m tuiModel) detailView() string {
	api := m.filtered[m.cursor]
	var b strings.Builder
	b.WriteString(m.headerView(api.Name) + "\n\n")
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
	b.WriteString("\n" + tuiMuted.Render(" r run  c copy  esc back  q quit"))
	return b.String()
}

func (m tuiModel) formView() string {
	api := m.filtered[m.cursor]
	var b strings.Builder
	b.WriteString(m.headerView("run "+api.Name) + "\n\n")
	b.WriteString(methodStyle(api.Method).Render(api.Method) + "  " + api.Name + "\n\n")
	if len(m.fields) == 0 {
		b.WriteString(tuiMuted.Render(" No parameters required. Press enter to execute.") + "\n")
	} else {
		b.WriteString(tuiMuted.Render(" PARAMETERS") + "\n")
		for i, f := range m.fields {
			label := f.label
			if f.required {
				label += " *"
			}
			prefix := "  "
			style := tuiMuted
			if i == m.fieldIndex {
				prefix = "› "
				style = tuiActive
			}
			value := f.value
			if f.isBody && value == "" {
				value = tuiMuted.Render("(JSON body)")
			}
			if i == m.fieldIndex {
				value = value + "█"
			}
			b.WriteString(prefix + style.Render(fmt.Sprintf("%-16s", label)) + " " + value + "\n")
		}
	}
	b.WriteString("\n" + tuiMuted.Render(" enter execute  esc back  q quit"))
	return b.String()
}

func (m tuiModel) bottomView() string {
	if m.executing {
		header := tuiBorder.Render(strings.Repeat("─", max(m.width, 40)))
		return header + "\n" + tuiActive.Render("Executing request...") + "\n"
	}
	if m.response == nil {
		return ""
	}
	r := m.response
	var b strings.Builder
	b.WriteString(tuiBorder.Render(strings.Repeat("─", max(m.width, 40))) + "\n")
	if r.Error != nil {
		b.WriteString(tuiError.Render("Error: "+r.Error.Error()) + "\n")
		return b.String()
	}
	b.WriteString(fmt.Sprintf("Status: %s   Timing: %s\n", r.Status, r.Timing))
	b.WriteString(tuiMuted.Render("Headers:") + "\n")
	for key, values := range r.Headers {
		for _, value := range values {
			b.WriteString(fmt.Sprintf("  %s: %s\n", key, value))
		}
	}
	b.WriteString("\n" + tuiMuted.Render("Response:") + "\n")
	body := string(r.Body)
	if len(r.Body) > 0 {
		var pretty strings.Builder
		if err := jsonIndent(&pretty, r.Body); err == nil {
			body = pretty.String()
		}
	}
	if body == "" {
		body = tuiMuted.Render("(empty)")
	}
	b.WriteString(body + "\n")
	if m.copyStatus != "" {
		style := tuiMuted
		if strings.HasPrefix(m.copyStatus, "Copy failed") {
			style = tuiError
		} else {
			style = tuiGet
		}
		b.WriteString("\n" + style.Render(m.copyStatus) + "\n")
	}
	return b.String()
}

func (m tuiModel) headerView(title string) string {
	left := tuiTitle.Render(" bench ")
	right := tuiMuted.Render(" " + m.project.Name + " ")
	middle := tuiHeader.Render(" " + title + " ")
	return left + middle + right
}

func (m *tuiModel) copyResponse() {
	if m.response == nil {
		return
	}
	data := m.response.Body
	if len(data) > 0 {
		var pretty strings.Builder
		if err := jsonIndent(&pretty, data); err == nil {
			data = []byte(pretty.String())
		}
	}
	if err := copyToClipboard(data); err != nil {
		m.copyStatus = "Copy failed: " + err.Error()
		return
	}
	m.copyStatus = "Copied response to clipboard"
}

func copyToClipboard(data []byte) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		if _, err := exec.LookPath("wl-copy"); err == nil {
			cmd = exec.Command("wl-copy")
		} else {
			cmd = exec.Command("xclip", "-selection", "clipboard")
		}
	default:
		return fmt.Errorf("clipboard not supported on %s", runtime.GOOS)
	}
	cmd.Stdin = strings.NewReader(string(data))
	return cmd.Run()
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

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func jsonIndent(dst *strings.Builder, data []byte) error {
	var v interface{}
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	pretty, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	dst.Write(pretty)
	return nil
}
