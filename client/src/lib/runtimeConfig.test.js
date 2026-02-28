import assert from "node:assert/strict";
import test from "node:test";
import { buildApiURL, buildWebSocketURL, toAmountCents } from "./runtimeConfig.js";

test("buildApiURL returns relative path when no base url is set", () => {
  assert.equal(buildApiURL("/api/me"), "/api/me");
});

test("buildApiURL joins base url and path without duplicate slashes", () => {
  assert.equal(buildApiURL("/api/me", { apiBaseURL: "https://api.example.com/" }), "https://api.example.com/api/me");
});

test("toAmountCents converts dollars to integer cents", () => {
  assert.equal(toAmountCents(12.5), 1250);
});

test("toAmountCents rejects zero or negative values", () => {
  assert.throws(() => toAmountCents(0), /amount must be greater than zero/);
  assert.throws(() => toAmountCents(-1), /amount must be greater than zero/);
});

test("buildWebSocketURL uses ws base url from env when provided", () => {
  assert.equal(buildWebSocketURL({ wsBaseURL: "wss://ws.example.com" }), "wss://ws.example.com/ws");
  assert.equal(buildWebSocketURL({ wsBaseURL: "wss://ws.example.com/ws" }), "wss://ws.example.com/ws");
});

test("buildWebSocketURL derives websocket url from api base url", () => {
  assert.equal(buildWebSocketURL({ apiBaseURL: "https://api.example.com" }), "wss://api.example.com/ws");
  assert.equal(buildWebSocketURL({ apiBaseURL: "http://localhost:8080/" }), "ws://localhost:8080/ws");
});

test("buildWebSocketURL falls back to browser location", () => {
  const wsURL = buildWebSocketURL({ locationProtocol: "https:", locationHost: "chat.example.com" });
  assert.equal(wsURL, "wss://chat.example.com/ws");
});
