package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type getResult struct {
	broadcast Broadcast
	err       error
}

type fakeYouTube struct {
	broadcast   Broadcast
	transitions []string
	getResults  []getResult
	getCalls    int
}

func (f *fakeYouTube) List(context.Context) ([]Broadcast, error) {
	return []Broadcast{f.broadcast}, nil
}
func (f *fakeYouTube) Get(context.Context, string) (Broadcast, error) {
	if f.getCalls < len(f.getResults) {
		result := f.getResults[f.getCalls]
		f.getCalls++
		return result.broadcast, result.err
	}
	return f.broadcast, nil
}
func (f *fakeYouTube) Transition(_ context.Context, _ string, target string) error {
	f.transitions = append(f.transitions, target)
	f.broadcast.Status = target
	return nil
}
func noWait(context.Context) error { return nil }

func TestControlsRequireProxyAuthenticatedUser(t *testing.T) {
	server := newServer(func(context.Context) (YouTubeClient, error) { return &fakeYouTube{}, nil }, &bytes.Buffer{}, noWait)
	for _, test := range []struct {
		name, actor string
		status      int
	}{{name: "missing actor", status: http.StatusUnauthorized}, {name: "authenticated user", actor: "viewer", status: http.StatusOK}} {
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

func TestConfiguredClientOriginAllowsCredentialedCrossOriginRequests(t *testing.T) {
	t.Setenv("CLIENT_URL", "https://client.example")
	server := newServer(func(context.Context) (YouTubeClient, error) { return &fakeYouTube{}, nil }, &bytes.Buffer{}, noWait)

	for _, path := range []string{"/controls/broadcasts", "/broadcasting"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodOptions, path, nil)
			req.Header.Set("Origin", "https://client.example")
			req.Header.Set("Access-Control-Request-Method", http.MethodPost)
			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)
			if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "https://client.example" {
				t.Errorf("Access-Control-Allow-Origin = %q", got)
			}
			if got := rec.Header().Get("Access-Control-Allow-Credentials"); got != "true" {
				t.Errorf("Access-Control-Allow-Credentials = %q", got)
			}
		})
	}

	req := httptest.NewRequest(http.MethodOptions, "/controls/broadcasts", nil)
	req.Header.Set("Origin", "https://other.example")
	req.Header.Set("Access-Control-Request-Method", http.MethodGet)
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if got := rec.Header().Get("Access-Control-Allow-Origin"); got != "" {
		t.Fatalf("unexpected Access-Control-Allow-Origin = %q", got)
	}
}

func TestStartRequiresActiveStreamThenReturnsObservedLiveStatus(t *testing.T) {
	fake := &fakeYouTube{broadcast: Broadcast{ID: "abc", Status: "ready", StreamStatus: "active"}}
	server := newServer(func(context.Context) (YouTubeClient, error) { return fake, nil }, &bytes.Buffer{}, noWait)
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

func TestStartAllowsTestingBroadcastButRejectsTestStarting(t *testing.T) {
	for _, test := range []struct {
		name       string
		status     string
		wantStatus int
	}{
		{name: "testing is startable", status: "testing", wantStatus: http.StatusOK},
		{name: "test starting is transitional", status: "testStarting", wantStatus: http.StatusConflict},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeYouTube{broadcast: Broadcast{ID: "abc", Status: test.status, StreamStatus: "active"}}
			server := newServer(func(context.Context) (YouTubeClient, error) { return fake, nil }, &bytes.Buffer{}, noWait)
			req := httptest.NewRequest(http.MethodPost, "/controls/broadcasts/abc/start", nil)
			req.Header.Set("X-Forwarded-User", "operator")
			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)
			if rec.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d: %s", rec.Code, test.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestStopRecordsSafeJSONAuditEvent(t *testing.T) {
	audit := &bytes.Buffer{}
	fake := &fakeYouTube{broadcast: Broadcast{ID: "abc", Status: "live", StreamStatus: "active"}}
	server := newServer(func(context.Context) (YouTubeClient, error) { return fake, nil }, audit, noWait)
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

func TestPostTransitionReadFailureIsNotReportedAsTimeout(t *testing.T) {
	audit := &bytes.Buffer{}
	fake := &fakeYouTube{getResults: []getResult{
		{broadcast: Broadcast{ID: "abc", Status: "ready", StreamStatus: "active"}},
		{err: errors.New("YouTube read failed")},
	}}
	server := newServer(func(context.Context) (YouTubeClient, error) { return fake, nil }, audit, noWait)
	req := httptest.NewRequest(http.MethodPost, "/controls/broadcasts/abc/start", nil)
	req.Header.Set("X-Forwarded-User", "operator")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadGateway {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusBadGateway, rec.Body.String())
	}
	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["error"] != "YouTube broadcast could not be read after transition" {
		t.Fatalf("error = %q", body["error"])
	}
	var event auditEvent
	if err := json.Unmarshal(audit.Bytes(), &event); err != nil {
		t.Fatalf("audit is not JSON: %v", err)
	}
	if event.Outcome != "failed" {
		t.Fatalf("outcome = %q, want failed", event.Outcome)
	}
}

func TestTransitionTimeoutRemainsDistinctFromReadFailure(t *testing.T) {
	audit := &bytes.Buffer{}
	fake := &fakeYouTube{getResults: []getResult{
		{broadcast: Broadcast{ID: "abc", Status: "ready", StreamStatus: "active"}},
		{broadcast: Broadcast{ID: "abc", Status: "ready", StreamStatus: "active"}},
		{broadcast: Broadcast{ID: "abc", Status: "testing", StreamStatus: "active"}},
		{broadcast: Broadcast{ID: "abc", Status: "liveStarting", StreamStatus: "active"}},
	}}
	server := newServer(func(context.Context) (YouTubeClient, error) { return fake, nil }, audit, noWait)
	req := httptest.NewRequest(http.MethodPost, "/controls/broadcasts/abc/start", nil)
	req.Header.Set("X-Forwarded-User", "operator")
	rec := httptest.NewRecorder()
	server.ServeHTTP(rec, req)
	if rec.Code != http.StatusGatewayTimeout {
		t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusGatewayTimeout, rec.Body.String())
	}
	var event auditEvent
	if err := json.Unmarshal(audit.Bytes(), &event); err != nil {
		t.Fatalf("audit is not JSON: %v", err)
	}
	if event.Outcome != "timed_out" || event.ObservedStatus != "liveStarting" {
		t.Fatalf("unexpected audit event: %#v", event)
	}
}

func TestInvalidBroadcastActionsAreAudited(t *testing.T) {
	for _, test := range []struct {
		name      string
		path      string
		action    string
		broadcast string
		target    string
	}{
		{name: "invalid broadcast ID", path: "/controls/broadcasts/bad.id/start", action: "start", broadcast: "bad.id", target: "live"},
		{name: "invalid action", path: "/controls/broadcasts/abc/delete", action: "delete", broadcast: "abc"},
	} {
		t.Run(test.name, func(t *testing.T) {
			audit := &bytes.Buffer{}
			server := newServer(func(context.Context) (YouTubeClient, error) { return &fakeYouTube{}, nil }, audit, noWait)
			req := httptest.NewRequest(http.MethodPost, test.path, nil)
			req.Header.Set("X-Forwarded-User", "operator")
			req.Header.Set("X-Request-ID", "invalid-request")
			rec := httptest.NewRecorder()
			server.ServeHTTP(rec, req)
			if rec.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want %d: %s", rec.Code, http.StatusBadRequest, rec.Body.String())
			}
			var event auditEvent
			if err := json.Unmarshal(audit.Bytes(), &event); err != nil {
				t.Fatalf("audit is not JSON: %v", err)
			}
			if event.RequestID != "invalid-request" || event.ActorID != "operator" || event.Action != test.action || event.BroadcastID != test.broadcast || event.TargetStatus != test.target || event.Outcome != "rejected" {
				t.Fatalf("unexpected audit event: %#v", event)
			}
		})
	}
}
