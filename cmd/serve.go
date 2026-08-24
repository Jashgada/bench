package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/spf13/cobra"
)

var serveAddress string

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Run the local Petstore mock server",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		server := &http.Server{Addr: serveAddress, Handler: newPetsHandler()}
		fmt.Fprintf(cmd.OutOrStdout(), "Petstore mock server listening at http://%s\n", serveAddress)
		fmt.Fprintln(cmd.OutOrStdout(), "Press Ctrl-C to stop.")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			return fmt.Errorf("serve mock API: %w", err)
		}
		return nil
	},
}

func init() {
	serveCmd.Flags().StringVarP(&serveAddress, "address", "a", "localhost:8080", "Address to listen on")
	rootCmd.AddCommand(serveCmd)
}

type pet struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
	Tag  string `json:"tag,omitempty"`
}

type petsStore struct {
	sync.Mutex
	pets   map[int]pet
	nextID int
}

func newPetsHandler() http.Handler {
	store := &petsStore{
		pets: map[int]pet{
			1: {ID: 1, Name: "Fido", Tag: "dog"},
			2: {ID: 2, Name: "Whiskers", Tag: "cat"},
		},
		nextID: 3,
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/pets" {
			handlePetsCollection(w, r, store)
			return
		}
		if strings.HasPrefix(r.URL.Path, "/pets/") {
			handlePet(w, r, store, strings.TrimPrefix(r.URL.Path, "/pets/"))
			return
		}
		http.NotFound(w, r)
	})
}

func handlePetsCollection(w http.ResponseWriter, r *http.Request, store *petsStore) {
	switch r.Method {
	case http.MethodGet:
		store.Lock()
		result := make([]pet, 0, len(store.pets))
		for _, item := range store.pets {
			if tag := r.URL.Query().Get("tag"); tag != "" && item.Tag != tag {
				continue
			}
			result = append(result, item)
		}
		if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
			if limit, err := strconv.Atoi(limitStr); err == nil && limit >= 0 && limit < len(result) {
				result = result[:limit]
			}
		}
		store.Unlock()
		writeJSON(w, http.StatusOK, result)
	case http.MethodPost:
		var input struct {
			Name string `json:"name"`
			Tag  string `json:"tag"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || strings.TrimSpace(input.Name) == "" {
			http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
			return
		}
		store.Lock()
		created := pet{ID: store.nextID, Name: input.Name, Tag: input.Tag}
		store.nextID++
		store.pets[created.ID] = created
		store.Unlock()
		writeJSON(w, http.StatusCreated, created)
	default:
		w.Header().Set("Allow", "GET, POST")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func handlePet(w http.ResponseWriter, r *http.Request, store *petsStore, idText string) {
	id, err := strconv.Atoi(idText)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	switch r.Method {
	case http.MethodGet:
		store.Lock()
		item, ok := store.pets[id]
		store.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		writeJSON(w, http.StatusOK, item)
	case http.MethodPut:
		var input struct {
			Name string `json:"name"`
			Tag  string `json:"tag"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil || strings.TrimSpace(input.Name) == "" {
			http.Error(w, `{"error":"name is required"}`, http.StatusBadRequest)
			return
		}
		store.Lock()
		if _, ok := store.pets[id]; !ok {
			store.Unlock()
			http.NotFound(w, r)
			return
		}
		updated := pet{ID: id, Name: input.Name, Tag: input.Tag}
		store.pets[id] = updated
		store.Unlock()
		writeJSON(w, http.StatusOK, updated)
	case http.MethodDelete:
		store.Lock()
		_, ok := store.pets[id]
		delete(store.pets, id)
		store.Unlock()
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		w.Header().Set("Allow", "GET, PUT, DELETE")
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
