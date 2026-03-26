import assert from "node:assert/strict";
import test from "node:test";

import {
  linkStoredMessageID,
  resolveSentMessageContent,
  saveSentMessageContent,
} from "./sentMessageContentCache.js";

function createStorage() {
  const values = new Map();
  return {
    getItem(key) {
      return values.has(key) ? values.get(key) : null;
    },
    setItem(key, value) {
      values.set(key, String(value));
    },
    removeItem(key) {
      values.delete(key);
    },
  };
}

test.beforeEach(() => {
  global.localStorage = createStorage();
});

test.afterEach(() => {
  delete global.localStorage;
});

test("saveSentMessageContent resolves by client id", () => {
  saveSentMessageContent({
    clientID: 11,
    to: "bob",
    contentKind: "text",
    body: "hello",
  });

  const resolved = resolveSentMessageContent({ clientID: 11 });
  assert.equal(resolved.body, "hello");
  assert.equal(resolved.content_kind, "text");
});

test("linkStoredMessageID makes cached content resolvable by server id", () => {
  saveSentMessageContent({
    clientID: 22,
    to: "alice",
    contentKind: "payment_request",
    body: "__microapp_v1__:{\"kind\":\"payment_request\"}",
  });

  linkStoredMessageID(22, 99);

  const resolved = resolveSentMessageContent({ serverID: 99 });
  assert.equal(resolved.client_id, 22);
  assert.equal(resolved.server_id, 99);
  assert.equal(resolved.content_kind, "payment_request");
});
