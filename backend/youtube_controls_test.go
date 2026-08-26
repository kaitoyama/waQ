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

func TestGoogleYouTubeClientListsActiveAndUpcomingPagesOnly(t *testing.T) {
	requests := map[string]int{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/youtube/v3/liveBroadcasts" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if r.URL.Query().Get("mine") != "true" {
			t.Errorf("mine = %q, want true", r.URL.Query().Get("mine"))
		}

		status, pageToken := r.URL.Query().Get("broadcastStatus"), r.URL.Query().Get("pageToken")
		requests[status+":"+pageToken]++
		w.Header().Set("Content-Type", "application/json")
		switch status + ":" + pageToken {
		case "active:":
			fmt.Fprint(w, `{"items":[{"id":"active","snippet":{"title":"Active"},"status":{"lifeCycleStatus":"live"}}],"nextPageToken":"active-next"}`)
		case "active:active-next":
			fmt.Fprint(w, `{"items":[{"id":"testing","snippet":{"title":"Testing"},"status":{"lifeCycleStatus":"testing"}}]}`)
		case "upcoming:":
			fmt.Fprint(w, `{"items":[{"id":"upcoming","snippet":{"title":"Upcoming"},"status":{"lifeCycleStatus":"ready"}}]}`)
		default:
			t.Fatalf("unexpected broadcastStatus/pageToken: %q/%q", status, pageToken)
		}
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

	if len(requests) != 3 || requests["active:"] != 1 || requests["active:active-next"] != 1 || requests["upcoming:"] != 1 {
		t.Fatalf("requests = %#v, want paginated active and upcoming requests only", requests)
	}
	if len(broadcasts) != 3 || broadcasts[0].ID != "active" || broadcasts[1].ID != "testing" || broadcasts[2].ID != "upcoming" {
		t.Fatalf("broadcasts = %#v, want active and upcoming broadcasts", broadcasts)
	}
}
