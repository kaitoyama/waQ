# YouTube Live Controls (DB-less) Specification

## Context

waQ creates YouTube broadcasts but does not control their lifecycle. Authentication is provided by a reverse proxy that strips any client-supplied identity header, authenticates the request, and injects `X-Forwarded-User`. The backend is reachable only through that proxy.

## Goals

- Permit only allowed proxy-authenticated users to start and stop a selected YouTube broadcast.
- Remove the insecure browser-visible `NEXT_PUBLIC_SECRET` / `X-Private-Key` authorization path.
- Do not add a database, persisted action history, OAuth login flow, or frontend dependency.
- Emit safe JSON structured audit events to stdout for every action attempt and outcome.

## Required behavior

### Authorization

- Read the actor from `X-Forwarded-User`.
- Reject empty or absent headers with HTTP 401.
- Authorize only actors in the comma-separated `WAQ_OPERATOR_USERS` environment variable; reject others with HTTP 403.
- Document that the backend must not be directly exposed and the proxy must strip then set the identity header.

### YouTube operations

- List controllable broadcasts and their title, video/broadcast ID, URL, lifecycle status, and bound-stream status.
- Start: only transition to `live` after confirming the bound stream is `active` and the broadcast is not already live or transitioning.
- Stop: only transition to `complete` when the broadcast is live; present an explicit frontend confirmation first.
- Poll YouTube after a transition and return a final observed lifecycle status or a safe timeout/error response.
- Serialize in-process operations for the same broadcast and prevent UI double clicks while a request runs.

### Audit output

Each operation must write JSON to stdout, including at minimum:

```json
{
  "event": "youtube_broadcast_action",
  "request_id": "correlation-id",
  "actor_id": "X-Forwarded-User value",
  "action": "start or stop",
  "broadcast_id": "YouTube broadcast ID",
  "before_status": "status before attempt",
  "target_status": "live or complete",
  "observed_status": "final status if available",
  "outcome": "requested, succeeded, rejected, failed, or timed_out"
}
```

Never log OAuth access/refresh tokens, stream keys, cookies, Authorization headers, or request bodies containing secrets.

## UI

- Provide a `/control` route accessible through the existing frontend.
- Show the authenticated actor, broadcast title/URL/status, stream readiness, and action result in Japanese.
- Disable start/stop when current state makes the operation invalid, but enforce the same rule server-side.
- Stop requires a dialog that identifies the target and requires explicit confirmation.

## Validation

- Strict TDD: test first and execute the expected failing test before implementation for each backend behavior.
- `go test ./...`, `go test -race ./...`, `npm --prefix front/waq run lint`, `npm --prefix front/waq run build`, and `git diff --check` must pass.
- Validate with a fake YouTube client; do not call a production broadcast during automated tests.
- Create a draft PR to `master` with tests and implementation evidence.
