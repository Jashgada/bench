package cmd

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPetsHandlerCRUD(t *testing.T) {
	server := httptest.NewServer(newPetsHandler())
	defer server.Close()

	response, err := http.Get(server.URL + "/pets?tag=dog")
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("list status: %s", response.Status)
	}
	var pets []pet
	if err := json.NewDecoder(response.Body).Decode(&pets); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()
	if len(pets) != 1 || pets[0].Name != "Fido" {
		t.Fatalf("unexpected filtered pets: %#v", pets)
	}

	request, err := http.NewRequest(http.MethodPost, server.URL+"/pets", strings.NewReader(`{"name":"Biscuit","tag":"dog"}`))
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err = http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("create status: %s", response.Status)
	}
	var created pet
	if err := json.NewDecoder(response.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	response.Body.Close()

	response, err = http.Get(server.URL + "/pets/" + string(rune('0'+created.ID)))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK {
		t.Fatalf("get status: %s", response.Status)
	}
	_, _ = io.Copy(io.Discard, response.Body)
	response.Body.Close()
}
