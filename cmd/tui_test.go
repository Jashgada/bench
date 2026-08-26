package cmd

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestTUIFilteringAndNavigation(t *testing.T) {
	model := tuiModel{items: []API{
		{Name: "createPet", Method: "POST", Path: "/pets"},
		{Name: "listPets", Method: "GET", Path: "/pets"},
	}}
	model.filtered = append([]API(nil), model.items...)
	model.filter = "list"
	model.applyFilter()
	if len(model.filtered) != 1 || model.filtered[0].Name != "listPets" {
		t.Fatalf("unexpected filter results: %#v", model.filtered)
	}
	model.filter = ""
	model.applyFilter()
	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyDown})
	model = updated.(tuiModel)
	if model.cursor != 1 {
		t.Fatalf("cursor did not move: %d", model.cursor)
	}
	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(tuiModel)
	if model.state != stateDetail {
		t.Fatal("enter did not open details")
	}
}

func TestTUIFormBuilding(t *testing.T) {
	model := tuiModel{
		items: []API{
			{Name: "updatePet", Method: "PUT", Path: "/pets/{id}", PathParams: []Parameter{{Name: "id", Required: true}}, RequestBodySchema: []byte(`{"type":"object"}`), RequestBodyRequired: true},
		},
	}
	model.filtered = append([]API(nil), model.items...)
	model.cursor = 0
	model.state = stateDetail
	model.buildForm()
	if len(model.fields) != 2 {
		t.Fatalf("expected 2 form fields, got %d", len(model.fields))
	}
	if model.fields[0].label != "id" || !model.fields[0].required {
		t.Fatalf("unexpected first field: %#v", model.fields[0])
	}
	if model.fields[1].label != "body" || !model.fields[1].isBody {
		t.Fatalf("unexpected second field: %#v", model.fields[1])
	}
}

func TestTUICommandBarSearch(t *testing.T) {
	model := tuiModel{items: []API{
		{Name: "createPet", Method: "POST", Path: "/pets"},
		{Name: "listPets", Method: "GET", Path: "/pets"},
		{Name: "getPet", Method: "GET", Path: "/pets/{id}"},
	}}
	model.filtered = append([]API(nil), model.items...)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("/")})
	model = updated.(tuiModel)
	if model.mode != cmdSearch {
		t.Fatal("slash did not enter search mode")
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("list")})
	model = updated.(tuiModel)
	if model.filter != "list" {
		t.Fatalf("filter not updated: %q", model.filter)
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(tuiModel)
	if model.mode != cmdNormal {
		t.Fatal("enter did not exit search mode")
	}
	if len(model.filtered) != 1 || model.filtered[0].Name != "listPets" {
		t.Fatalf("filter not applied: %#v", model.filtered)
	}
}

func TestTUICommandBarCommand(t *testing.T) {
	model := tuiModel{
		items: []API{
			{Name: "listPets", Method: "GET", Path: "/pets"},
		},
		project: Project{Name: "pets"},
	}
	model.filtered = append([]API(nil), model.items...)

	updated, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(":")})
	model = updated.(tuiModel)
	if model.mode != cmdCommand {
		t.Fatal("colon did not enter command mode")
	}

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("listPets")})
	model = updated.(tuiModel)

	updated, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = updated.(tuiModel)
	if model.mode != cmdNormal {
		t.Fatal("enter did not exit command mode")
	}
	if model.state != stateDetail {
		t.Fatal("command did not navigate to detail")
	}
}

func TestTUIProjectsCommand(t *testing.T) {
	model := tuiModel{project: Project{Name: "pets"}, items: []API{{Name: "listPets"}}}
	model.filtered = append([]API(nil), model.items...)
	updated, _ := model.executeCommand("projects")
	model = updated.(tuiModel)
	if model.state != stateProjects {
		t.Fatal("projects command did not open project picker")
	}
}

func TestTUIGroupCommand(t *testing.T) {
	model := tuiModel{items: []API{
		{Name: "listPets", Tags: []string{"pokemon"}},
		{Name: "listItems", Tags: []string{"items"}},
	}}
	model.filtered = append([]API(nil), model.items...)
	updated, _ := model.executeCommand("group pokemon")
	model = updated.(tuiModel)
	if model.tagFilter != "pokemon" || len(model.filtered) != 1 || model.filtered[0].Name != "listPets" {
		t.Fatalf("unexpected group result: %#v", model)
	}
	updated, _ = model.executeCommand("group all")
	model = updated.(tuiModel)
	if model.tagFilter != "" || len(model.filtered) != 2 {
		t.Fatalf("group all did not clear filter: %#v", model)
	}
}

func TestTUIResponseScrolling(t *testing.T) {
	model := tuiModel{
		state:    stateDetail,
		height:   12,
		response: &ResponseResult{Body: []byte("1\n2\n3\n4\n5\n6\n7\n8")},
	}
	model.scrollResponse(4)
	if model.responseScroll == 0 {
		t.Fatal("response did not scroll")
	}
	model.scrollResponse(-100)
	if model.responseScroll != 0 {
		t.Fatalf("response scrolled before beginning: %d", model.responseScroll)
	}
}
