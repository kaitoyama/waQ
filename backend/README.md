# waQ backend

## YouTube lifecycle controls

`GET /controls/broadcasts` lists broadcasts. `POST /controls/broadcasts/:id/start` and `POST /controls/broadcasts/:id/stop` change lifecycle state.

Every user authenticated by the trusted reverse proxy may operate broadcasts. The backend uses `X-Forwarded-User` only to identify the actor in responses and audit logs; it does not maintain an application-level operator allowlist.

Set `CLIENT_URL` to the exact frontend origin. The backend permits credentialed CORS requests only from that origin, and the frontend includes proxy session credentials on `/broadcasting` and `/controls` requests.

The backend trusts `X-Forwarded-User` only because it must be reachable exclusively through the authenticated reverse proxy. Configure that proxy to strip every client-supplied `X-Forwarded-User`, authenticate the request, and then set the header itself. Never expose the backend directly to the internet.

Each lifecycle action emits a JSON audit event to stdout. Audit output deliberately excludes OAuth tokens, stream keys, authorization headers, and request bodies.
