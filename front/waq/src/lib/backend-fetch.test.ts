import assert from "node:assert/strict"
import test from "node:test"

import { backendFetch } from "./backend-fetch.ts"

test("backendFetch includes browser credentials while preserving request options", async () => {
  const originalFetch = globalThis.fetch
  let capturedInput: RequestInfo | URL | undefined
  let capturedInit: RequestInit | undefined
  globalThis.fetch = (async (input: RequestInfo | URL, init?: RequestInit) => {
    capturedInput = input
    capturedInit = init
    return new Response(null, { status: 204 })
  }) as typeof fetch

  try {
    await backendFetch("https://backend.example/controls/broadcasts", { method: "POST" })
  } finally {
    globalThis.fetch = originalFetch
  }

  assert.equal(capturedInput, "https://backend.example/controls/broadcasts")
  assert.equal(capturedInit?.method, "POST")
  assert.equal(capturedInit?.credentials, "include")
})
