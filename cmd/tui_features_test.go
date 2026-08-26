package cmd

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSortAPIs(t *testing.T) {
	items := []API{
		{Name: "zulu", Method: "POST", Tags: []string{"pets"}},
		{Name: "alpha", Method: "GET", Tags: []string{}},
		{Name: "mike", Method: "DELETE", Tags: []string{"items"}},
	}
	tests := []struct {
		name string
		key  string
		asc  bool
		want []string
	}{
		{"by name asc", "name", true, []string{"alpha", "mike", "zulu"}},
		{"by name desc", "name", false, []string{"zulu", "mike", "alpha"}},
		{"by method asc", "method", true, []string{"mike", "alpha", "zulu"}},
		{"by method desc", "method", false, []string{"zulu", "alpha", "mike"}},
		{"by tag asc empty last", "tag", true, []string{"mike", "zulu", "alpha"}},
		{"by tag desc empty still last", "tag", false, []string{"zulu", "mike", "alpha"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := append([]API(nil), items...)
			sortAPIs(got, tt.key, tt.asc)
			var names []string
			for _, api := range got {
				names = append(names, api.Name)
			}
			for i := range tt.want {
				if names[i] != tt.want[i] {
					t.Fatalf("order = %v, want %v", names, tt.want)
				}
			}
		})
	}
}

func TestHistoryNavigationBounds(t *testing.T) {
	if got := olderHistoryIndex(-1, 3); got != 2 {
		t.Fatalf("older from live = %d, want 2", got)
	}
	if got := olderHistoryIndex(0, 3); got != 0 {
		t.Fatalf("older at oldest = %d, want 0", got)
	}
	if got := newerHistoryIndex(2, 3); got != -1 {
		t.Fatalf("newer at newest = %d, want -1 (live)", got)
	}
	if got := newerHistoryIndex(0, 3); got != 1 {
		t.Fatalf("newer from 0 = %d, want 1", got)
	}
	if got := olderHistoryIndex(-1, 0); got != -1 {
		t.Fatalf("empty history older = %d, want -1", got)
	}
}

func TestResponseSearchMatches(t *testing.T) {
	lines := []string{"alpha", "Beta test", "gamma beta", "delta"}
	matches := matchLines(lines, "beta")
	if len(matches) != 2 || matches[0] != 1 || matches[1] != 2 {
		t.Fatalf("matchLines = %v, want [1 2]", matches)
	}
	if got := matchLines(lines, ""); len(got) != 0 {
		t.Fatalf("empty query should have no matches: %v", got)
	}
	if got := nextMatchLine(matches, 1); got != 2 {
		t.Fatalf("next from 1 = %d, want 2", got)
	}
	if got := nextMatchLine(matches, 2); got != 1 {
		t.Fatalf("next wraps = %d, want 1", got)
	}
	if got := prevMatchLine(matches, 2); got != 1 {
		t.Fatalf("prev from 2 = %d, want 1", got)
	}
	if got := prevMatchLine(matches, 1); got != 2 {
		t.Fatalf("prev wraps = %d, want 2", got)
	}
	if got := matchRank(matches, 2); got != 2 {
		t.Fatalf("rank of line 2 = %d, want 2", got)
	}
}

func TestTUIHelpOverlay(t *testing.T) {
	model := tuiModel{project: Project{Name: "p"}, items: []API{{Name: "a"}}, sortKey: "name", sortAsc: true}
	model.filtered = model.items
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("?")})
	model = updated.(tuiModel)
	if !model.helpOverlay {
		t.Fatal("? did not open help overlay")
	}
	view := model.View()
	if view == "" {
		t.Fatal("help overlay rendered empty")
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("x")})
	model = updated.(tuiModel)
	if model.helpOverlay {
		t.Fatal("key press did not close help overlay")
	}
	updated, _ = model.executeCommand("help")
	model = updated.(tuiModel)
	if !model.helpOverlay {
		t.Fatal(":help did not open help overlay")
	}
}

func TestTUIUnknownCommandStatus(t *testing.T) {
	model := tuiModel{project: Project{Name: "p"}, items: []API{{Name: "listPets"}}, sortKey: "name", sortAsc: true}
	model.filtered = model.items
	updated, _ := model.executeCommand("zzz")
	model = updated.(tuiModel)
	want := "unknown command: zzz"
	if model.copyStatus != want {
		t.Fatalf("copyStatus = %q, want %q", model.copyStatus, want)
	}
}

func TestTUIHeaderScrolling(t *testing.T) {
	headers := make(map[string][]string)
	for i := 0; i < 20; i++ {
		headers[fmtHeaderKey(i)] = []string{"value"}
	}
	model := tuiModel{
		state:       stateDetail,
		height:      12,
		responseTab: responseHeaders,
		response:    &ResponseResult{Status: "200 OK", Headers: headers},
		sortKey:     "name",
	}
	model.scrollHeaders(10)
	if model.headerScroll == 0 {
		t.Fatal("headers did not scroll")
	}
	model.scrollHeaders(100)
	maxOffset := 20 - max(12/2-6, 3)
	if model.headerScroll != maxOffset {
		t.Fatalf("headerScroll = %d, want %d", model.headerScroll, maxOffset)
	}
	model.scrollHeaders(-100)
	if model.headerScroll != 0 {
		t.Fatalf("header scrolled before beginning: %d", model.headerScroll)
	}
}

func fmtHeaderKey(i int) string {
	return "X-Header-" + string(rune('A'+i))
}

func TestTUIHistoryNavigationKeys(t *testing.T) {
	model := tuiModel{
		project: Project{Name: "p"},
		items:   []API{{Name: "getPet", Method: "GET", Path: "/pets"}},
		state:   stateDetail,
		sortKey: "name",
		sortAsc: true,
		history: []historyEntry{
			{api: API{Name: "getPet"}, timestamp: time.Now()},
			{api: API{Name: "getPet"}, timestamp: time.Now()},
			{api: API{Name: "getPet"}, timestamp: time.Now()},
		},
		historyIndex: -1,
	}
	model.filtered = model.items

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	model = updated.(tuiModel)
	if model.historyIndex != 2 {
		t.Fatalf("[ from live = %d, want 2", model.historyIndex)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("[")})
	model = updated.(tuiModel)
	if model.historyIndex != 1 {
		t.Fatalf("[ again = %d, want 1", model.historyIndex)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	model = updated.(tuiModel)
	if model.historyIndex != 2 {
		t.Fatalf("] = %d, want 2", model.historyIndex)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("]")})
	model = updated.(tuiModel)
	if model.historyIndex != -1 {
		t.Fatalf("] past newest = %d, want -1", model.historyIndex)
	}
}

func TestTUIHistoryAppendOnResponse(t *testing.T) {
	model := tuiModel{
		project:      Project{Name: "p"},
		state:        stateDetail,
		pendingEntry: &historyEntry{api: API{Name: "getPet"}},
		sortKey:      "name",
		sortAsc:      true,
	}
	result := ResponseResult{Status: "200 OK", Body: []byte(`{"ok":true}`)}
	updated, _ := model.Update(responseMsg{result: result})
	model = updated.(tuiModel)
	if len(model.history) != 1 {
		t.Fatalf("history length = %d, want 1", len(model.history))
	}
	if model.history[0].result.Status != "200 OK" {
		t.Fatalf("stored result status = %q", model.history[0].result.Status)
	}
	if model.pendingEntry != nil || model.historyIndex != -1 {
		t.Fatal("pending entry not cleared or history index wrong")
	}
}

func TestTUISortKeybindings(t *testing.T) {
	model := tuiModel{
		project: Project{Name: "p"},
		items: []API{
			{Name: "zeta", Method: "POST"},
			{Name: "alpha", Method: "GET"},
			{Name: "mid", Method: "DELETE"},
		},
		filtered: []API{
			{Name: "zeta", Method: "POST"},
			{Name: "alpha", Method: "GET"},
			{Name: "mid", Method: "DELETE"},
		},
		state:   stateList,
		sortKey: "name",
		sortAsc: true,
	}
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("M")})
	model = updated.(tuiModel)
	if model.items[0].Method != "DELETE" || !model.sortAsc || model.sortKey != "method" {
		t.Fatalf("shift+m sort by method failed: %+v", model.items)
	}
	if model.cursor != 0 || model.viewport != 0 {
		t.Fatal("cursor/viewport not reset on sort change")
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("M")})
	model = updated.(tuiModel)
	if !(!model.sortAsc && model.items[0].Method == "POST") {
		t.Fatalf("second shift+m did not toggle direction: %+v", model.items)
	}
}

func TestTUIResponseSearchMode(t *testing.T) {
	model := tuiModel{
		project: Project{Name: "p"},
		items:   []API{{Name: "getPet"}},
		state:   stateDetail,
		height:  40,
		response: &ResponseResult{
			Status: "200 OK",
			Body:   []byte("{\"a\":\"find me\",\"b\":1,\"c\":\"find me too\"}"),
		},
		sortKey: "name",
		sortAsc: true,
	}
	model.filtered = model.items

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	model = updated.(tuiModel)
	if model.mode != cmdRespSearch {
		t.Fatal("/ in detail with body tab did not enter response search mode")
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("find me")})
	model = updated.(tuiModel)
	if model.respSearch != "find me" {
		t.Fatalf("respSearch = %q", model.respSearch)
	}
	if model.searchLine == 0 {
		t.Fatal("search did not jump to first match")
	}
	firstLine := model.searchLine

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(tuiModel)
	if model.mode != cmdRespSearch {
		t.Fatal("enter exited search mode; expected it to stay for cycling")
	}
	if model.searchLine <= firstLine {
		t.Fatalf("enter did not advance match: %d -> %d", firstLine, model.searchLine)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	model = updated.(tuiModel)
	if model.mode != cmdNormal {
		t.Fatal("esc did not exit search mode")
	}
	if model.respSearch == "" {
		t.Fatal("esc cleared the search query; n/N should still work")
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("N")})
	model = updated.(tuiModel)
	if model.searchLine != firstLine {
		t.Fatalf("N did not go to previous match: got %d, want %d", model.searchLine, firstLine)
	}
}
