import assert from "node:assert/strict";
import test from "node:test";

import { connectWebSocket } from "../ws.js";

class FakeWebSocket {
  constructor(url, subprotocols) {
    this.url = url;
    this.subprotocols = subprotocols;
    this.readyState = 0;
  }
}

test("connectWebSocket derives ws url and bearer subprotocol", () => {
  process.env.VITE_WS_URL = "wss://chat.example.com";
  process.env.VITE_API_BASE_URL = "";
  const originalWS = global.WebSocket;
  global.WebSocket = FakeWebSocket;

  try {
    const ws = connectWebSocket("token-123");
    assert.equal(ws.url, "wss://chat.example.com/ws");
    assert.deepEqual(ws.subprotocols, ["bearer.token-123"]);
  } finally {
    global.WebSocket = originalWS;
  }
});

test("connectWebSocket can derive from API base URL when WS base URL is absent", () => {
  process.env.VITE_WS_URL = "";
  process.env.VITE_API_BASE_URL = "https://api.example.com";
  const originalWS = global.WebSocket;
  global.WebSocket = FakeWebSocket;

  try {
    const ws = connectWebSocket(null);
    assert.equal(ws.url, "wss://api.example.com/ws");
    assert.equal(ws.subprotocols, undefined);
  } finally {
    global.WebSocket = originalWS;
  }
});
