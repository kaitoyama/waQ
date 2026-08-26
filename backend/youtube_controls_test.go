package main

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"google.golang.org/api/option"
	"google.golang.org/api/youtube/v3"
)

func TestGoogleYouTubeClientListsAllControllablePages(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/youtube/v3/liveBroadcasts" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("mine") != "true" {
			t.Errorf("mine = %q, want true", r.URL.Query().Get("mine"))
		}
		requests++
		w.Header().Set("Content-Type", "application/json")
		if r.URL.Query().Get("pageToken") == "next" {
			fmt.Fprint(w, `{"items":[{"id":"upcoming","snippet":{"title":"Upcoming"},"status":{"lifeCycleStatus":"ready"}},{"id":"revoked-history","snippet":{"title":"Revoked"},"status":{"lifeCycleStatus":"revoked"}}]}`)
			return
		}
		fmt.Fprint(w, `{"items":[{"id":"active","snippet":{"title":"Active"},"status":{"lifeCycleStatus":"live"}},{"id":"completed-history","snippet":{"title":"Completed"},"status":{"lifeCycleStatus":"complete"}}],"nextPageToken":"next"}`)
	}))
	defer server.Close()

	service, err := youtube.NewService(context.Background(), option.WithEndpoint(server.URL+"/"), option.WithoutAuthentication())
	if err != nil {
		t.Fatal(err)
	}
	client := &googleYouTubeClient{service: service}
	broadcasts, err := client.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if requests != 2 {
		t.Fatalf("requests = %d, want every page", requests)
	}
	if len(broadcasts) != 2 || broadcasts[0].ID != "active" || broadcasts[1].ID != "upcoming" {
		t.Fatalf("broadcasts = %#v, want current/upcoming broadcasts from every page", broadcasts)
	}
}
