package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"sync"

	"github.com/labstack/echo/v4"
)

type Broadcast struct {
	ID           string `json:"id"`
	Title        string `json:"title"`
	URL          string `json:"url"`
	Status       string `json:"status"`
	StreamStatus string `json:"streamStatus"`
}
type YouTubeClient interface {
	List(context.Context) ([]Broadcast, error)
	Get(context.Context, string) (Broadcast, error)
	Transition(context.Context, string, string) error
}
type clientFactory func(context.Context) (YouTubeClient, error)
type waiter func(context.Context) error

type controlApp struct {
	factory   clientFactory
	operators map[string]struct{}
	audit     io.Writer
	wait      waiter
	locks     keyedLocker
	auditMu   sync.Mutex
}
type keyedLocker struct {
	mu    sync.Mutex
	locks map[string]*keyedLock
}
type keyedLock struct {
	mu   sync.Mutex
	refs int
}

func (l *keyedLocker) lock(key string) func() {
	l.mu.Lock()
	if l.locks == nil {
		l.locks = map[string]*keyedLock{}
	}
	entry := l.locks[key]
	if entry == nil {
		entry = &keyedLock{}
		l.locks[key] = entry
	}
	entry.refs++
	l.mu.Unlock()
	entry.mu.Lock()
	return func() {
		entry.mu.Unlock()
		l.mu.Lock()
		entry.refs--
		if entry.refs == 0 {
			delete(l.locks, key)
		}
		l.mu.Unlock()
	}
}

func newServer(factory clientFactory, operatorUsers string, audit io.Writer, wait waiter) *echo.Echo {
	app := &controlApp{factory: factory, operators: parseOperators(operatorUsers), audit: audit, wait: wait}
	e := echo.New()
	e.GET("/controls/broadcasts", app.list)
	e.POST("/controls/broadcasts/:id/:action", app.action)
	return e
}
func parseOperators(users string) map[string]struct{} {
	operators := map[string]struct{}{}
	for _, user := range strings.Split(users, ",") {
		if user = strings.TrimSpace(user); user != "" {
			operators[user] = struct{}{}
		}
	}
	return operators
}
func (a *controlApp) actor(c echo.Context) (string, error) {
	actor := strings.TrimSpace(c.Request().Header.Get("X-Forwarded-User"))
	if actor == "" {
		return "", echo.NewHTTPError(http.StatusUnauthorized, "proxy authentication is required")
	}
	if _, ok := a.operators[actor]; !ok {
		return actor, echo.NewHTTPError(http.StatusForbidden, "operator authorization is required")
	}
	return actor, nil
}
func (a *controlApp) list(c echo.Context) error {
	actor, err := a.actor(c)
	if err != nil {
		return err
	}
	client, err := a.factory(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusServiceUnavailable, "YouTube client is unavailable")
	}
	broadcasts, err := client.List(c.Request().Context())
	if err != nil {
		return echo.NewHTTPError(http.StatusBadGateway, "YouTube broadcasts could not be listed")
	}
	return c.JSON(http.StatusOK, map[string]any{"actor": actor, "broadcasts": broadcasts})
}

var broadcastID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)

func (a *controlApp) action(c echo.Context) error {
	id, action := c.Param("id"), c.Param("action")
	if !broadcastID.MatchString(id) || (action != "start" && action != "stop") {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid broadcast action")
	}
	actor, err := a.actor(c)
	if err != nil {
		a.writeAudit(auditEvent{ActorID: actor, Action: action, BroadcastID: id, TargetStatus: targetFor(action), Outcome: "rejected", RequestID: requestID(c.Request())})
		return err
	}
	unlock := a.locks.lock(id)
	defer unlock()
	client, err := a.factory(c.Request().Context())
	if err != nil {
		a.writeAudit(auditEvent{ActorID: actor, Action: action, BroadcastID: id, TargetStatus: targetFor(action), Outcome: "failed", RequestID: requestID(c.Request())})
		return echo.NewHTTPError(http.StatusServiceUnavailable, "YouTube client is unavailable")
	}
	broadcast, err := client.Get(c.Request().Context(), id)
	if err != nil {
		a.writeAudit(auditEvent{ActorID: actor, Action: action, BroadcastID: id, TargetStatus: targetFor(action), Outcome: "failed", RequestID: requestID(c.Request())})
		return echo.NewHTTPError(http.StatusBadGateway, "YouTube broadcast could not be read")
	}
	target := targetFor(action)
	event := auditEvent{ActorID: actor, Action: action, BroadcastID: id, BeforeStatus: broadcast.Status, TargetStatus: target, RequestID: requestID(c.Request())}
	if err := transitionAllowed(action, broadcast); err != nil {
		event.ObservedStatus, event.Outcome = broadcast.Status, "rejected"
		a.writeAudit(event)
		return c.JSON(http.StatusConflict, map[string]string{"error": err.Error(), "status": broadcast.Status})
	}
	if err := client.Transition(c.Request().Context(), id, target); err != nil {
		event.Outcome = "failed"
		a.writeAudit(event)
		return echo.NewHTTPError(http.StatusBadGateway, "YouTube transition failed")
	}
	for attempt := 0; attempt < 3; attempt++ {
		observed, err := client.Get(c.Request().Context(), id)
		if err != nil {
			break
		}
		event.ObservedStatus = observed.Status
		if observed.Status == target {
			event.Outcome = "succeeded"
			a.writeAudit(event)
			return c.JSON(http.StatusOK, observed)
		}
		if attempt < 2 && a.wait != nil {
			if err := a.wait(c.Request().Context()); err != nil {
				break
			}
		}
	}
	event.Outcome = "timed_out"
	a.writeAudit(event)
	return c.JSON(http.StatusGatewayTimeout, map[string]string{"error": "YouTube transition timed out", "status": event.ObservedStatus})
}
func targetFor(action string) string {
	if action == "start" {
		return "live"
	}
	return "complete"
}
func transitionAllowed(action string, broadcast Broadcast) error {
	if action == "start" {
		if broadcast.Status == "live" || broadcast.Status == "testing" || broadcast.Status == "liveStarting" {
			return fmt.Errorf("broadcast is already live or transitioning")
		}
		if broadcast.StreamStatus != "active" {
			return fmt.Errorf("bound stream is not active")
		}
		return nil
	}
	if broadcast.Status != "live" {
		return fmt.Errorf("broadcast is not live")
	}
	return nil
}

type auditEvent struct {
	Event          string `json:"event"`
	RequestID      string `json:"request_id"`
	ActorID        string `json:"actor_id"`
	Action         string `json:"action"`
	BroadcastID    string `json:"broadcast_id"`
	BeforeStatus   string `json:"before_status"`
	TargetStatus   string `json:"target_status"`
	ObservedStatus string `json:"observed_status"`
	Outcome        string `json:"outcome"`
}

func (a *controlApp) writeAudit(event auditEvent) {
	event.Event = "youtube_broadcast_action"
	a.auditMu.Lock()
	defer a.auditMu.Unlock()
	_ = json.NewEncoder(a.audit).Encode(event)
}
func requestID(r *http.Request) string {
	if value := strings.TrimSpace(r.Header.Get("X-Request-ID")); value != "" {
		return value
	}
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err == nil {
		return hex.EncodeToString(bytes)
	}
	return "unavailable"
}
