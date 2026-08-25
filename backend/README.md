# waQ backend

## YouTube lifecycle controls

`GET /controls/broadcasts` lists broadcasts. `POST /controls/broadcasts/:id/start` and `POST /controls/broadcasts/:id/stop` change lifecycle state.

Set `WAQ_OPERATOR_USERS` to a comma-separated allowlist, for example `WAQ_OPERATOR_USERS=alice@example.com,bob@example.com`.

The backend trusts `X-Forwarded-User` only because it must be reachable exclusively through the authenticated reverse proxy. Configure that proxy to strip every client-supplied `X-Forwarded-User`, authenticate the request, and then set the header itself. Never expose the backend directly to the internet.

Each lifecycle action emits a JSON audit event to stdout. Audit output deliberately excludes OAuth tokens, stream keys, authorization headers, and request bodies.
