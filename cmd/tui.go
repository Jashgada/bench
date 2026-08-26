package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type tuiState int

const (
	stateList tuiState = iota
	stateDetail
	stateForm
	stateProjects
	stateEnvs
)

type cmdMode int

const (
	cmdNormal cmdMode = iota
	cmdSearch
	cmdCommand
	cmdRespSearch
)

type responseTab int

const (
	responseBody responseTab = iota
	responseHeaders
)

type formField struct {
	label          string
	value          string
	required       bool
	isBody         bool
	isBodyProperty bool
	isRawBody      bool
	bodyType       string
}

type responseMsg struct {
	result ResponseResult
}

type projectMsg struct {
	project Project
	err     error
}

type clearCopyMsg struct{}

type tuiModel struct {
	project       Project
	projects      []string
	projectCursor int
	envs          []string
	envCursor     int
	envName       string
	items         []API
	filtered      []API
	cursor        int
	viewport      int
	filter        string
	tagFilter     string
	state         tuiState
	width         int
	height        int

	fields       []formField
	fieldIndex   int
	rawBody      bool
	rawBodyInput string

	response       *ResponseResult
	responseScroll int
	headerScroll   int
	responseTab    responseTab
	executing      bool
	copyStatus     string

	history      []historyEntry
	pendingEntry *historyEntry
	historyIndex int

	sortKey string
	sortAsc bool

	respSearch string
	searchLine int

	helpOverlay bool

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
	model := tuiModel{project: project, items: items, filtered: items, sortKey: "name", sortAsc: true, historyIndex: -1}
	_, err = tea.NewProgram(model, tea.WithAltScreen()).Run()
	return err
}

func (m tuiModel) Init() tea.Cmd { return nil }

func (m tuiModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case responseMsg:
		m.executing = false
		m.response = &msg.result
		m.responseScroll = 0
		m.headerScroll = 0
		m.searchLine = 0
		m.responseTab = responseBody
		m.state = stateDetail
		if m.pendingEntry != nil {
			m.pendingEntry.result = msg.result
			m.pendingEntry.timestamp = time.Now()
			m.history = append(m.history, *m.pendingEntry)
			m.pendingEntry = nil
		}
		m.historyIndex = -1
		return m, nil
	case projectMsg:
		if msg.err != nil {
			m.copyStatus = msg.err.Error()
			return m, nil
		}
		m.setProject(msg.project)
		m.copyStatus = "Project reloaded"
		return m, nil
	case clearCopyMsg:
		m.copyStatus = ""
		return m, nil
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		if m.state == stateForm && m.rawBody {
			return m.updateRawBody(msg)
		}
		if m.state == stateForm {
			return m.updateForm(msg)
		}
		if m.mode != cmdNormal {
			return m.updateCommandBar(msg)
		}
		return m.updateNormal(msg)
	}
	return m, nil
}

func (m tuiModel) updateForm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.state = stateDetail
	case "up", "shift+tab":
		if m.fieldIndex > 0 {
			m.fieldIndex--
		}
	case "down", "tab":
		if m.fieldIndex < len(m.fields)-1 {
			m.fieldIndex++
		}
	case "enter":
		if !m.formValid() {
			m.copyStatus = "Required fields must be filled"
			return m, nil
		}
		return m, m.submitForm()
	case "backspace":
		field := &m.fields[m.fieldIndex]
		if len(field.value) > 0 {
			field.value = field.value[:len(field.value)-1]
		}
	case ".":
		if m.hasRequestBody() && (m.fields[m.fieldIndex].isBodyProperty || m.fields[m.fieldIndex].isBody) {
			m.rawBody = true
			m.rawBodyInput = ""
		}
	case "ctrl+c":
		return m, tea.Quit
	default:
		for _, r := range msg.String() {
			if r >= ' ' && r <= '~' {
				m.fields[m.fieldIndex].value += string(r)
			}
		}
	}
	return m, nil
}

func (m tuiModel) updateCommandBar(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = cmdNormal
		m.input = ""
	case "enter":
		if m.mode == cmdRespSearch {
			m.jumpSearch(1)
			return m, nil
		}
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
			switch m.mode {
			case cmdSearch:
				m.filter = m.input
				m.applyFilter()
			case cmdRespSearch:
				m.respSearch = m.input
				m.liveSearchJump()
			}
		}
	case "ctrl+c":
		return m, tea.Quit
	default:
		s := msg.String()
		for _, r := range s {
			if r >= ' ' && r <= '~' {
				m.input += string(r)
				switch m.mode {
				case cmdSearch:
					m.filter = m.input
					m.applyFilter()
				case cmdRespSearch:
					m.respSearch = m.input
					m.liveSearchJump()
				}
			}
		}
	}
	return m, nil
}

func (m tuiModel) updateNormal(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.helpOverlay {
		m.helpOverlay = false
		return m, nil
	}
	if m.state == stateDetail && m.displayedResult() != nil && !m.executing {
		switch msg.String() {
		case "tab", "left", "right":
			if m.responseTab == responseBody {
				m.responseTab = responseHeaders
			} else {
				m.responseTab = responseBody
			}
			m.responseScroll = 0
			m.headerScroll = 0
			return m, nil
		}
	}
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "?":
		m.helpOverlay = true
		return m, nil
	case "esc":
		switch m.state {
		case stateProjects:
			m.state = stateList
		case stateEnvs:
			m.state = stateList
		case stateForm:
			m.state = stateDetail
		case stateDetail:
			m.state = stateList
			m.response = nil
			m.historyIndex = -1
		case stateList:
			m.filter = ""
			m.applyFilter()
		}
	case "up", "k":
		if m.state == stateDetail && m.displayedResult() != nil && !m.executing {
			if m.responseTab == responseHeaders {
				m.scrollHeaders(-1)
			} else {
				m.scrollResponse(-1)
			}
			return m, nil
		}
		switch m.state {
		case stateProjects:
			if m.projectCursor > 0 {
				m.projectCursor--
			}
		case stateEnvs:
			if m.envCursor > 0 {
				m.envCursor--
			}
		case stateList:
			if m.cursor > 0 {
				m.cursor--
				m.ensureCursorVisible()
			}
		case stateForm:
			if m.fieldIndex > 0 {
				m.fieldIndex--
			}
		}
	case "down", "j":
		if m.state == stateDetail && m.displayedResult() != nil && !m.executing {
			if m.responseTab == responseHeaders {
				m.scrollHeaders(1)
			} else {
				m.scrollResponse(1)
			}
			return m, nil
		}
		switch m.state {
		case stateProjects:
			if m.projectCursor < len(m.projects)-1 {
				m.projectCursor++
			}
		case stateEnvs:
			if m.envCursor < len(m.envs)-1 {
				m.envCursor++
			}
		case stateList:
			if m.cursor < len(m.filtered)-1 {
				m.cursor++
				m.ensureCursorVisible()
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
		case stateProjects:
			if len(m.projects) > 0 {
				m.switchProject(m.projects[m.projectCursor])
			}
		case stateEnvs:
			if len(m.envs) > 0 {
				m.switchEnvironment(m.envs[m.envCursor])
			}
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
			if !m.formValid() {
				m.copyStatus = "Required fields must be filled"
				return m, nil
			}
			return m, m.submitForm()
		}
	case "r":
		if m.state == stateDetail && m.historyIndex >= 0 {
			return m.rerunHistory()
		}
		if m.state == stateDetail && len(m.filtered) > 0 && !m.executing {
			m.buildForm()
			m.state = stateForm
		}
	case ".":
		if m.state == stateForm && m.hasRequestBody() {
			m.rawBody = true
			m.rawBodyInput = ""
		}
	case "c":
		if m.state == stateDetail && m.displayedResult() != nil {
			m.copyResponse()
			return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearCopyMsg{} })
		}
	case "y":
		if m.state == stateDetail && len(m.filtered) > 0 {
			m.copyCurl()
			return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearCopyMsg{} })
		}
	case "[":
		if m.state == stateDetail && len(m.history) > 0 {
			m.historyOlder()
			return m, nil
		}
	case "]":
		if m.state == stateDetail && len(m.history) > 0 {
			m.historyNewer()
			return m, nil
		}
	case "n":
		if m.state == stateDetail && m.responseTab == responseBody && m.respSearch != "" {
			m.jumpSearch(1)
			return m, nil
		}
	case "N":
		if m.state == stateDetail && m.responseTab == responseBody && m.respSearch != "" {
			m.jumpSearch(-1)
			return m, nil
		}
		if m.state == stateList {
			return m.applySort("name")
		}
	case "M":
		if m.state == stateList {
			return m.applySort("method")
		}
	case "T":
		if m.state == stateList {
			return m.applySort("tag")
		}
	case "/":
		if m.state == stateDetail && m.responseTab == responseBody && m.displayedResult() != nil {
			m.mode = cmdRespSearch
			m.input = m.respSearch
			return m, nil
		}
		m.mode = cmdSearch
		m.input = m.filter
	case ":":
		m.mode = cmdCommand
		m.input = ""
	}
	return m, nil
}

func (m *tuiModel) applySort(key string) (tea.Model, tea.Cmd) {
	if m.sortKey == key {
		m.sortAsc = !m.sortAsc
	} else {
		m.sortKey = key
		m.sortAsc = true
	}
	sortAPIs(m.items, m.sortKey, m.sortAsc)
	m.applyFilter()
	m.cursor = 0
	m.viewport = 0
	return *m, nil
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
	case "group":
		if len(parts) != 2 {
			m.copyStatus = "Usage: :group <tag>"
			return m, nil
		}
		m.tagFilter = ""
		if parts[1] != "all" {
			m.tagFilter = parts[1]
		}
		m.state = stateList
		m.applyFilter()
		return m, nil
	case "reload":
		return m, m.reloadProject()
	case "update":
		return m, m.updateProject()
	case "projects", "p":
		m.openProjects()
		return m, nil
	case "project":
		if len(parts) != 2 {
			return m, nil
		}
		m.switchProject(parts[1])
		return m, nil
	case "envs":
		m.openEnvs()
		return m, nil
	case "env":
		if len(parts) != 2 {
			m.copyStatus = "Usage: :env <name> (or :envs to browse)"
			return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearCopyMsg{} })
		}
		m.switchEnvironment(parts[1])
		return m, nil
	case "curl":
		return m.runCurlCommand(strings.TrimSpace(strings.TrimPrefix(cmd, "curl")))
	case "help", "h":
		m.helpOverlay = true
		return m, nil
	case "butts":
		return "poopy"
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
				m.historyIndex = -1
				return m, nil
			}
		}
		m.copyStatus = "unknown command: " + parts[0]
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearCopyMsg{} })
	}
}

func (m *tuiModel) buildForm() {
	m.buildFormFor(m.filtered[m.cursor])
}

func (m *tuiModel) buildFormFor(api API) {
	m.fields = nil
	m.fieldIndex = 0
	m.rawBody = false
	m.rawBodyInput = ""
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
		properties := bodyProperties(api.RequestBodySchema)
		if len(properties) == 0 {
			m.fields = append(m.fields, formField{label: "body", isBody: true, required: api.RequestBodyRequired, isRawBody: true})
		} else {
			for _, property := range properties {
				m.fields = append(m.fields, formField{label: property.name, required: property.required, isBodyProperty: true, bodyType: property.kind})
			}
		}
	}
}

func (m *tuiModel) openProjects() {
	projects, err := projectNames()
	if err != nil {
		return
	}
	m.projects = projects
	m.projectCursor = 0
	for i, name := range projects {
		if name == m.project.Name {
			m.projectCursor = i
			break
		}
	}
	m.state = stateProjects
}

func (m *tuiModel) switchProject(name string) {
	project, err := loadProject(name)
	if err != nil {
		return
	}
	m.setProject(project)
	_ = setCurrentProject(name)
}

func (m *tuiModel) openEnvs() {
	envs, err := listEnvironments(m.project.Name)
	if err != nil {
		m.copyStatus = err.Error()
		return
	}
	m.envs = envs
	m.envCursor = 0
	for i, name := range envs {
		if name == m.envName {
			m.envCursor = i
			break
		}
	}
	m.state = stateEnvs
}

func (m *tuiModel) switchEnvironment(name string) {
	if _, err := loadEnvironment(m.project.Name, name); err != nil {
		m.copyStatus = fmt.Sprintf("environment %q not found", name)
		return
	}
	if err := setCurrentEnvironment(m.project.Name, name); err != nil {
		m.copyStatus = err.Error()
		return
	}
	m.envName = name
	m.state = stateList
	m.copyStatus = "Switched to environment " + name
}

func (m tuiModel) runCurlCommand(input string) (tea.Model, tea.Cmd) {
	if input == "" {
		m.copyStatus = "Usage: :curl <curl command>"
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearCopyMsg{} })
	}
	request, err := parseCurl(input)
	if err != nil {
		m.copyStatus = "Curl parse failed: " + err.Error()
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return clearCopyMsg{} })
	}
	api := API{Name: "curl", Method: request.Method, Path: request.URL}
	for _, header := range request.Headers {
		api.Headers = append(api.Headers, Parameter{Name: header[0]})
	}
	params := map[string]string{}
	for i, header := range request.Headers {
		params[header[0]] = request.Headers[i][1]
	}
	project := Project{Name: m.project.Name}
	m.executing = true
	body := []byte(request.Body)
	return m, func() tea.Msg {
		return responseMsg{result: executeRequest(project, api, params, body)}
	}
}

func (m *tuiModel) setProject(project Project) {
	items := append([]API(nil), project.APIs...)
	sortKey := m.sortKey
	if sortKey == "" {
		sortKey = "name"
	}
	sortAPIs(items, sortKey, m.sortAsc || m.sortKey == "")
	m.project = project
	m.items = items
	m.filtered = items
	m.cursor = 0
	m.viewport = 0
	m.filter = ""
	m.tagFilter = ""
	m.response = nil
	m.history = nil
	m.pendingEntry = nil
	m.historyIndex = -1
	m.respSearch = ""
	m.searchLine = 0
	m.envName = currentEnvironmentName(project.Name)
	m.state = stateList
}

func (m tuiModel) reloadProject() tea.Cmd {
	name := m.project.Name
	return func() tea.Msg {
		project, err := loadProject(name)
		return projectMsg{project: project, err: err}
	}
}

func (m tuiModel) updateProject() tea.Cmd {
	name := m.project.Name
	return func() tea.Msg {
		project, err := refreshProjectFromSource(name)
		return projectMsg{project: project, err: err}
	}
}

func formRequest(fields []formField) (map[string]string, []byte, error) {
	params := map[string]string{}
	var body []byte
	for _, f := range fields {
		if f.isRawBody {
			body = []byte(f.value)
		} else if f.isBodyProperty {
			continue
		} else {
			params[f.label] = f.value
		}
	}
	if body == nil {
		properties := map[string]json.RawMessage{}
		hasBody := false
		for _, f := range fields {
			if f.isBodyProperty {
				hasBody = true
				if f.value != "" {
					value, err := encodeBodyValue(f.value, f.bodyType)
					if err != nil {
						return nil, nil, fmt.Errorf("invalid %s: %w", f.label, err)
					}
					properties[f.label] = value
				}
			}
		}
		if hasBody {
			encoded, err := json.Marshal(properties)
			if err != nil {
				return nil, nil, err
			}
			body = encoded
		}
	}
	return params, body, nil
}

func (m tuiModel) submitForm() tea.Cmd {
	api := m.filtered[m.cursor]
	params, body, err := formRequest(m.fields)
	if err != nil {
		return func() tea.Msg { return responseMsg{result: ResponseResult{Error: err}} }
	}
	m.pendingEntry = &historyEntry{api: api, params: params, body: body}
	m.executing = true
	project := m.project
	return func() tea.Msg {
		result := executeRequest(project, api, params, body)
		return responseMsg{result: result}
	}
}

func (m tuiModel) formValid() bool {
	for _, field := range m.fields {
		if field.required && strings.TrimSpace(field.value) == "" {
			return false
		}
	}
	return true
}

func (m tuiModel) updateRawBody(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", ".":
		m.rawBody = false
	case "enter":
		if strings.TrimSpace(m.rawBodyInput) == "" {
			if m.currentAPI().RequestBodyRequired {
				m.copyStatus = "Request body is required"
				return m, nil
			}
		} else if !json.Valid([]byte(m.rawBodyInput)) {
			m.copyStatus = "Invalid JSON body"
			return m, nil
		}
		m.copyStatus = ""
		return m, m.submitRawBody()
	case "backspace":
		if len(m.rawBodyInput) > 0 {
			m.rawBodyInput = m.rawBodyInput[:len(m.rawBodyInput)-1]
		}
	case "ctrl+c":
		return m, tea.Quit
	default:
		for _, r := range msg.String() {
			if r >= ' ' && r <= '~' {
				m.rawBodyInput += string(r)
			}
		}
	}
	return m, nil
}

func (m tuiModel) submitRawBody() tea.Cmd {
	api := m.currentAPI()
	project := m.project
	body := []byte(m.rawBodyInput)
	m.pendingEntry = &historyEntry{api: api, body: append([]byte(nil), body...)}
	m.executing = true
	return func() tea.Msg { return responseMsg{result: executeRequest(project, api, nil, body)} }
}

func (m tuiModel) currentAPI() API { return m.filtered[m.cursor] }

func (m tuiModel) hasRequestBody() bool { return len(m.currentAPI().RequestBodySchema) > 0 }

func (m *tuiModel) applyFilter() {
	needle := strings.ToLower(m.filter)
	m.filtered = make([]API, 0, len(m.items))
	for _, api := range m.items {
		matchesTag := m.tagFilter == ""
		for _, tag := range api.Tags {
			if strings.EqualFold(tag, m.tagFilter) {
				matchesTag = true
				break
			}
		}
		if matchesTag && (needle == "" || strings.Contains(strings.ToLower(api.Name+" "+api.Method+" "+api.Path+" "+strings.Join(api.Tags, " ")), needle)) {
			m.filtered = append(m.filtered, api)
		}
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	if m.cursor < 0 {
		m.cursor = 0
	}
	m.viewport = 0
}

func (m *tuiModel) ensureCursorVisible() {
	rows := m.visibleRows()
	if m.cursor < m.viewport {
		m.viewport = m.cursor
	}
	if m.cursor >= m.viewport+rows {
		m.viewport = m.cursor - rows + 1
	}
}

func (m tuiModel) visibleRows() int {
	if m.height <= 0 {
		return 10
	}
	return max(m.height-8, 3)
}

func (m tuiModel) View() string {
	var content string
	switch m.state {
	case stateForm:
		content = m.formView()
	case stateProjects:
		content = m.projectsView()
	case stateEnvs:
		content = m.envsView()
	case stateDetail:
		content = m.detailView()
	default:
		content = m.listView()
	}

	if m.width > 4 {
		content = tuiPanel.Width(m.width - 4).Render(content)
	} else {
		content = tuiPanel.Render(content)
	}
	if m.response != nil || m.executing {
		content = lipgloss.JoinVertical(lipgloss.Left, content, m.bottomView())
	}

	if m.helpOverlay {
		return helpOverlayView(m)
	}
	return lipgloss.JoinVertical(lipgloss.Left, m.statusBar(), m.commandBar(), content)
}

func (m tuiModel) statusBar() string {
	project := tuiTitle.Render(" bench ") + " " + m.project.Name
	baseURL := tuiMuted.Render("server: ") + m.project.BaseURL
	count := tuiMuted.Render(fmt.Sprintf("operations: %d", len(m.items)))
	status := project + "  " + baseURL + "  " + count
	if m.envName != "" {
		status += "  " + tuiActive.Render("env: "+m.envName)
	}
	return status
}

func (m tuiModel) envsView() string {
	var b strings.Builder
	b.WriteString(m.headerView("Environments") + "\n\n")
	if len(m.envs) == 0 {
		b.WriteString(tuiMuted.Render(" No environments found. Create one with: bench env add <name> --set key=value") + "\n")
	} else {
		for i, name := range m.envs {
			line := "  " + name
			if name == m.envName {
				line += tuiActive.Render(" (active)")
			}
			if i == m.envCursor {
				line = tuiSelected.Render("› " + strings.TrimPrefix(line, "  "))
			}
			b.WriteString(line + "\n")
		}
	}
	b.WriteString("\n" + tuiMuted.Render(" enter switch  esc back"))
	return b.String()
}

func (m tuiModel) projectsView() string {
	var b strings.Builder
	b.WriteString(m.headerView("Projects") + "\n\n")
	if len(m.projects) == 0 {
		b.WriteString(tuiMuted.Render(" No projects found.") + "\n")
	} else {
		for i, name := range m.projects {
			line := "  " + name
			if name == m.project.Name {
				line += tuiMuted.Render(" (current)")
			}
			if i == m.projectCursor {
				line = tuiSelected.Render("› " + strings.TrimPrefix(line, "  "))
			}
			b.WriteString(line + "\n")
		}
	}
	return b.String()
}

func (m tuiModel) commandBar() string {
	prompt := ">"
	text := m.input
	switch m.mode {
	case cmdSearch:
		prompt = "/"
		text = m.input
	case cmdRespSearch:
		prompt = "/"
		text = m.input
	case cmdCommand:
		prompt = ":"
		text = m.input
	}
	bar := tuiPrompt.Render(prompt) + " " + text
	if m.mode != cmdNormal {
		bar += "█"
	} else if m.copyStatus != "" {
		bar += "  " + tuiMuted.Render(m.copyStatus)
	}
	if m.width > 4 {
		return tuiCommand.Width(m.width - 4).Render(bar)
	}
	return tuiCommand.Render(bar)
}

func (m tuiModel) listView() string {
	var b strings.Builder
	title := "APIs"
	if m.tagFilter != "" {
		title += " / " + m.tagFilter
	}
	b.WriteString(m.headerView(title) + "\n")
	b.WriteString(tuiMuted.Render("  METHOD   PATH                                      OPERATION                 TAG") + "\n")
	end := min(m.viewport+m.visibleRows(), len(m.filtered))
	for i := m.viewport; i < end; i++ {
		api := m.filtered[i]
		method := methodStyle(api.Method).Render(fmt.Sprintf("%-8s", api.Method))
		line := fmt.Sprintf("  %s %-42s %-24s %s", method, api.Path, api.Name, tuiMuted.Render(strings.Join(api.Tags, ",")))
		if i == m.cursor {
			line = tuiSelected.Render("› " + strings.TrimPrefix(line, "  "))
		}
		b.WriteString(line + "\n")
	}
	direction := "asc"
	if !m.sortAsc {
		direction = "desc"
	}
	b.WriteString(fmt.Sprintf("\n"+tuiMuted.Render(" showing %d-%d of %d  ·  sort: %s %s"), displayStart(m.viewport, len(m.filtered)), end, len(m.filtered), m.sortKey, direction))
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func displayStart(offset, count int) int {
	if count == 0 {
		return 0
	}
	return offset + 1
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
	if m.historyIndex >= 0 && m.historyIndex < len(m.history) {
		b.WriteString("\n" + tuiActive.Render(fmt.Sprintf("history %d/%d", m.historyIndex+1, len(m.history))) + "\n")
	}
	if m.historyIndex >= 0 {
		b.WriteString("\n" + tuiMuted.Render(" r re-run this request  [ older  ] newer  esc back"))
	} else {
		b.WriteString("\n" + tuiMuted.Render(" r run  c copy  y curl  / search  ? help  esc back  q quit"))
	}
	return b.String()
}

func (m tuiModel) formView() string {
	api := m.filtered[m.cursor]
	var b strings.Builder
	b.WriteString(m.headerView("run "+api.Name) + "\n\n")
	b.WriteString(methodStyle(api.Method).Render(api.Method) + "  " + api.Name + "\n\n")
	if m.rawBody {
		b.WriteString(tuiMuted.Render(" RAW JSON BODY") + "\n\n")
		b.WriteString(m.rawBodyInput + "█\n")
		b.WriteString("\n" + tuiMuted.Render(" enter execute  esc return to fields  q quit"))
		return b.String()
	}
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
	b.WriteString("\n" + tuiMuted.Render(" enter execute  . raw JSON  esc back  q quit"))
	if m.copyStatus != "" {
		b.WriteString("\n" + tuiError.Render(m.copyStatus))
	}
	return b.String()
}

func (m tuiModel) bottomView() string {
	if m.executing {
		header := tuiBorder.Render(strings.Repeat("─", max(m.width, 40)))
		return header + "\n" + tuiActive.Render("Executing request...") + "\n"
	}
	r := m.displayedResult()
	if r == nil {
		return ""
	}
	var b strings.Builder
	b.WriteString(tuiBorder.Render(strings.Repeat("─", max(m.width, 40))) + "\n")
	if r.Error != nil {
		b.WriteString(tuiError.Render("Error: "+r.Error.Error()) + "\n")
		return b.String()
	}
	if m.historyIndex >= 0 && m.historyIndex < len(m.history) {
		entry := m.history[m.historyIndex]
		b.WriteString(tuiMuted.Render(fmt.Sprintf("history %d/%d  executed %s", m.historyIndex+1, len(m.history), entry.timestamp.Format("15:04:05"))) + "\n")
	}
	b.WriteString(fmt.Sprintf("Status: %s   Timing: %s\n", r.Status, r.Timing))
	for _, warning := range r.Warnings {
		b.WriteString(tuiPost.Render("warning: "+warning) + "\n")
	}
	visible := max(m.height/2-6, 3)
	if m.responseTab == responseHeaders {
		b.WriteString("\n" + tuiActive.Render("Headers") + "   " + tuiMuted.Render("Response body") + "\n")
		lines := headerLines(r.Headers)
		start := min(m.headerScroll, max(len(lines)-visible, 0))
		end := min(start+visible, len(lines))
		if len(lines) == 0 {
			b.WriteString(tuiMuted.Render("  (no headers)") + "\n")
		} else {
			b.WriteString(strings.Join(lines[start:end], "\n") + "\n")
		}
		if len(lines) > visible {
			b.WriteString(tuiMuted.Render(fmt.Sprintf("header lines %d-%d of %d", start+1, end, len(lines))) + "\n")
		}
		hint := "tab/left/right: response body"
		if len(lines) > visible {
			hint += "  j/k scroll"
		}
		b.WriteString("\n" + tuiMuted.Render(hint) + "\n")
		return b.String()
	}
	tabsLine := "\n" + tuiActive.Render("Response body") + "   " + tuiMuted.Render("Headers")
	lines := displayLines(r)
	if m.respSearch != "" {
		matches := matchLines(lines, m.respSearch)
		rank := matchRank(matches, m.searchLine)
		count := fmt.Sprintf("match %d/%d", rank, len(matches))
		if len(matches) == 0 {
			tabsLine += "   " + tuiError.Render(count)
		} else {
			tabsLine += "   " + tuiActive.Render(count)
		}
	}
	b.WriteString(tabsLine + "\n")
	bodyLines := lines
	if len(bodyLines) == 0 {
		bodyLines = []string{tuiMuted.Render("(empty)")}
	}
	start := min(m.responseScroll, max(len(bodyLines)-visible, 0))
	end := min(start+visible, len(bodyLines))
	b.WriteString(strings.Join(bodyLines[start:end], "\n") + "\n")
	if len(bodyLines) > visible {
		b.WriteString(tuiMuted.Render(fmt.Sprintf("response lines %d-%d of %d", start+1, end, len(bodyLines))) + "\n")
	}
	hint := "tab/left/right: headers"
	if len(bodyLines) > visible {
		hint += "  j/k scroll"
	}
	if m.respSearch != "" {
		hint += "  enter/n next  N prev"
	}
	b.WriteString("\n" + tuiMuted.Render(hint) + "\n")
	if m.copyStatus != "" {
		style := tuiMuted
		if strings.HasPrefix(m.copyStatus, "Copy failed") || strings.HasPrefix(m.copyStatus, "Curl failed") {
			style = tuiError
		} else {
			style = tuiGet
		}
		b.WriteString("\n" + style.Render(m.copyStatus) + "\n")
	}
	return b.String()
}

func displayLines(r *ResponseResult) []string {
	if r == nil || len(r.Body) == 0 {
		return nil
	}
	var pretty strings.Builder
	if err := jsonIndent(&pretty, r.Body); err == nil {
		return strings.Split(pretty.String(), "\n")
	}
	return strings.Split(string(r.Body), "\n")
}

func headerLines(headers http.Header) []string {
	keys := make([]string, 0, len(headers))
	for key := range headers {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	var lines []string
	for _, key := range keys {
		for _, value := range headers[key] {
			lines = append(lines, "  "+key+": "+value)
		}
	}
	return lines
}

func (m *tuiModel) scrollResponse(delta int) {
	r := m.displayedResult()
	if r == nil {
		return
	}
	lines := displayLines(r)
	visible := max(m.height/2-6, 3)
	maxOffset := max(len(lines)-visible, 0)
	m.responseScroll += delta
	if m.responseScroll < 0 {
		m.responseScroll = 0
	}
	if m.responseScroll > maxOffset {
		m.responseScroll = maxOffset
	}
}

func (m *tuiModel) scrollHeaders(delta int) {
	r := m.displayedResult()
	if r == nil {
		return
	}
	count := 0
	for _, values := range r.Headers {
		count += len(values)
	}
	visible := max(m.height/2-6, 3)
	maxOffset := max(count-visible, 0)
	m.headerScroll += delta
	if m.headerScroll < 0 {
		m.headerScroll = 0
	}
	if m.headerScroll > maxOffset {
		m.headerScroll = maxOffset
	}
}

func (m *tuiModel) jumpSearch(dir int) {
	r := m.displayedResult()
	if r == nil || m.respSearch == "" {
		return
	}
	matches := matchLines(displayLines(r), m.respSearch)
	if len(matches) == 0 {
		m.searchLine = 0
		return
	}
	if dir >= 0 {
		m.searchLine = nextMatchLine(matches, m.searchLine)
	} else {
		m.searchLine = prevMatchLine(matches, m.searchLine)
	}
	m.ensureMatchVisible()
}

func (m *tuiModel) liveSearchJump() {
	r := m.displayedResult()
	if r == nil || m.respSearch == "" {
		return
	}
	matches := matchLines(displayLines(r), m.respSearch)
	if len(matches) == 0 {
		return
	}
	m.searchLine = firstMatchAtOrAfter(matches, m.responseScroll)
	m.ensureMatchVisible()
}

func (m *tuiModel) ensureMatchVisible() {
	visible := max(m.height/2-6, 3)
	if m.searchLine < m.responseScroll {
		m.responseScroll = m.searchLine
	}
	if m.searchLine >= m.responseScroll+visible {
		m.responseScroll = m.searchLine - visible + 1
	}
}

func (m tuiModel) headerView(title string) string {
	left := tuiTitle.Render(" bench ")
	right := tuiMuted.Render(" " + m.project.Name + " ")
	middle := tuiHeader.Render(" " + title + " ")
	return left + middle + right
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
