package cmd

import (
	"encoding/json"
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type historyEntry struct {
	api       API
	params    map[string]string
	body      []byte
	result    ResponseResult
	timestamp time.Time
}

func olderHistoryIndex(current, length int) int {
	if length <= 0 {
		return current
	}
	if current < 0 {
		return length - 1
	}
	if current > 0 {
		return current - 1
	}
	return 0
}

func newerHistoryIndex(current, length int) int {
	if length <= 0 || current < 0 {
		return -1
	}
	if current >= length-1 {
		return -1
	}
	return current + 1
}

func (m *tuiModel) displayedResult() *ResponseResult {
	if m.historyIndex >= 0 && m.historyIndex < len(m.history) {
		return &m.history[m.historyIndex].result
	}
	return m.response
}

func (m *tuiModel) historyOlder() {
	m.historyIndex = olderHistoryIndex(m.historyIndex, len(m.history))
	m.resetDetailScroll()
}

func (m *tuiModel) historyNewer() {
	m.historyIndex = newerHistoryIndex(m.historyIndex, len(m.history))
	m.resetDetailScroll()
}

func (m *tuiModel) resetDetailScroll() {
	m.responseScroll = 0
	m.headerScroll = 0
	m.searchLine = 0
}

func stringifyBodyValue(v interface{}) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	default:
		data, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprint(t)
		}
		return string(data)
	}
}

func (m *tuiModel) rerunHistory() (tea.Model, tea.Cmd) {
	if m.historyIndex < 0 || m.historyIndex >= len(m.history) {
		return *m, nil
	}
	entry := m.history[m.historyIndex]
	m.buildFormFor(entry.api)
	for i := range m.fields {
		if v, ok := entry.params[m.fields[i].label]; ok {
			m.fields[i].value = v
		}
	}
	m.rawBody = false
	m.rawBodyInput = ""
	if len(entry.body) > 0 && !m.fillBodyFieldsFromJSON(entry.body) {
		m.rawBodyInput = string(entry.body)
		m.pendingEntry = &historyEntry{api: entry.api, params: entry.params, body: entry.body}
		m.executing = true
		project := m.project
		return *m, func() tea.Msg {
			return responseMsg{result: executeRequest(project, entry.api, entry.params, entry.body)}
		}
	}
	return *m, m.submitForm()
}

func (m *tuiModel) fillBodyFieldsFromJSON(body []byte) bool {
	var obj map[string]interface{}
	if err := json.Unmarshal(body, &obj); err != nil {
		return false
	}
	hasProps := false
	for _, f := range m.fields {
		if f.isBodyProperty {
			hasProps = true
			break
		}
	}
	if !hasProps {
		return false
	}
	for i := range m.fields {
		if !m.fields[i].isBodyProperty {
			continue
		}
		if v, ok := obj[m.fields[i].label]; ok {
			m.fields[i].value = stringifyBodyValue(v)
		}
	}
	return true
}
