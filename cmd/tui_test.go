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
	if !model.detail {
		t.Fatal("enter did not open details")
	}
}
