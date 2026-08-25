package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type fakeYouTube struct {
	broadcast   Broadcast
	transitions []string
}

func (f *fakeYouTube) List(context.Context) ([]Broadcast, error) {
	return []Broadcast{f.broadcast}, nil
}
func (f *fakeYouTube) Get(context.Context, string) (Broadcast, error) { return f.broadcast, nil }
func (f *fakeYouTube) Transition(_ context.Context, _ string, target string) error {
	f.transitions = append(f.transitions, target)
	f.broadcast.Status = target
	return nil
}
func noWait(context.Context) error { return nil }

func TestControlsRequireProxyAuthenticatedOperator(t *testing.T) {
	server := newServer(func(context.Context) (YouTubeClient, error) { return &fakeYouTube{}, nil }, "operator", &bytes.Buffer{}, noWait)
	for _, test := range []struct {
		name, actor string
		status      int
	}{{name: "missing actor", status: http.StatusUnauthorized}, {name: "non operator", actor: "viewer", status: http.StatusForbidden}, {name: "operator", actor: "operator", status: http.StatusOK}} {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/controls/broadcasts", nil)
			req.Header.Set("X-Forwarded-User", test.actor)
			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)
			if rec.Code != test.status {
				t.Fatalf("status = %d, want %d: %s", rec.Code, test.status, rec.Body.String())
			}
		})
	}
}

func TestStartRequiresActiveStreamThenReturnsObservedLiveStatus(t *testing.T) {
	fake := &fakeYouTube{broadcast: Broadcast{ID: "abc", Status: "ready", StreamStatus: "active"}}
	server := newServer(func(context.Context) (YouTubeClient, error) { return fake, nil }, "operator", &bytes.Buffer{}, noWait)
	req := httptest.NewRequest(http.MethodPost, "/controls/broadcasts/abc/start", nil)
	req.Header.Set("X-Forwarded-User", "operator")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	if len(fake.transitions) != 1 || fake.transitions[0] != "live" {
		t.Fatalf("transitions = %#v, want [live]", fake.transitions)
	}
}

func TestStopRecordsSafeJSONAuditEvent(t *testing.T) {
	audit := &bytes.Buffer{}
	fake := &fakeYouTube{broadcast: Broadcast{ID: "abc", Status: "live", StreamStatus: "active"}}
	server := newServer(func(context.Context) (YouTubeClient, error) { return fake, nil }, "operator", audit, noWait)
	req := httptest.NewRequest(http.MethodPost, "/controls/broadcasts/abc/stop", nil)
	req.Header.Set("X-Forwarded-User", "operator")
	req.Header.Set("X-Request-ID", "request-123")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusOK, rec.Body.String())
	}
	var event auditEvent
	if err := json.Unmarshal(audit.Bytes(), &event); err != nil {
		t.Fatalf("audit is not JSON: %v", err)
	}
	if event.Event != "youtube_broadcast_action" || event.RequestID != "request-123" || event.ActorID != "operator" || event.Action != "stop" || event.BroadcastID != "abc" || event.BeforeStatus != "live" || event.TargetStatus != "complete" || event.ObservedStatus != "complete" || event.Outcome != "succeeded" {
		t.Fatalf("unexpected audit event: %#v", event)
	}
}
