package activity

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEarlyResolver(t *testing.T) {
	activities := earlyActivitiesResponse{
		Activities: []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}{
			{ID: "act-1", Name: "Deep Engineering"},
			{ID: "act-2", Name: "Code Review"},
			{ID: "act-3", Name: "Admin / Meetings"},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/activities" {
			http.NotFound(w, r)
			return
		}
		if r.Header.Get("Authorization") != "Bearer test-token" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(activities)
	}))
	t.Cleanup(srv.Close)

	resolver := &EarlyResolver{
		baseURL: srv.URL,
		token:   "test-token",
		client:  srv.Client(),
	}

	t.Run("resolve known ID", func(t *testing.T) {
		name, err := resolver.Resolve(context.Background(), "act-1")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if name != "Deep Engineering" {
			t.Errorf("got %q, want %q", name, "Deep Engineering")
		}
	})

	t.Run("resolve another known ID", func(t *testing.T) {
		name, err := resolver.Resolve(context.Background(), "act-3")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if name != "Admin / Meetings" {
			t.Errorf("got %q, want %q", name, "Admin / Meetings")
		}
	})

	t.Run("unknown ID returns ErrNotFound", func(t *testing.T) {
		_, err := resolver.Resolve(context.Background(), "act-unknown")
		if err != ErrNotFound {
			t.Errorf("got err %v, want ErrNotFound", err)
		}
	})
}

func TestEarlyResolverCachesResults(t *testing.T) {
	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		resp := earlyActivitiesResponse{
			Activities: []struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			}{
				{ID: "a", Name: "Alpha"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	t.Cleanup(srv.Close)

	resolver := &EarlyResolver{
		baseURL: srv.URL,
		token:   "tok",
		client:  srv.Client(),
	}

	// call twice — server should only be hit once
	resolver.Resolve(context.Background(), "a")
	resolver.Resolve(context.Background(), "a")

	if callCount != 1 {
		t.Errorf("expected 1 HTTP call, got %d", callCount)
	}
}
