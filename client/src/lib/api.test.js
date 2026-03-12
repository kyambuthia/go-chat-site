import assert from "node:assert/strict";
import test from "node:test";

import {
  addContact,
  getContacts,
  getInbox,
  getMessageThreads,
  getOutbox,
  getWalletTransfers,
  markMessageDelivered,
  markMessageRead,
  removeContact,
  sendMoney,
  setAuthErrorHandler,
  setToken,
  syncMessages,
  updateMe,
} from "../api.js";

function jsonResponse(status, payload, headers = {}) {
  return {
    ok: status >= 200 && status < 300,
    status,
    headers: {
      get(name) {
        const key = Object.keys(headers).find((h) => h.toLowerCase() === String(name).toLowerCase());
        return key ? headers[key] : null;
      },
    },
    async json() {
      return payload;
    },
  };
}

test("getContacts sends Authorization header and uses VITE_API_BASE_URL", async () => {
  process.env.VITE_API_BASE_URL = "https://api.example.com";
  setToken("token-123");

  let capturedURL = "";
  let capturedOptions = null;
  global.fetch = async (url, options) => {
    capturedURL = url;
    capturedOptions = options;
    return jsonResponse(200, []);
  };

  await getContacts();

  assert.equal(capturedURL, "https://api.example.com/api/contacts");
  assert.equal(capturedOptions.headers.Authorization, "Bearer token-123");
});

test("sendMoney sends amount as integer cents", async () => {
  process.env.VITE_API_BASE_URL = "http://localhost:8080";
  setToken("token-xyz");

  let requestBody = "";
  global.fetch = async (_url, options) => {
    requestBody = options.body;
    return jsonResponse(200, { ok: true });
  };

  await sendMoney("bob", 12.34);

  const parsed = JSON.parse(requestBody);
  assert.equal(parsed.username, "bob");
  assert.equal(parsed.amount_cents, 1234);
});

test("addContact posts username to contacts endpoint", async () => {
  process.env.VITE_API_BASE_URL = "http://localhost:8080";
  setToken("token-add");

  let capturedURL = "";
  let capturedOptions = null;
  global.fetch = async (url, options) => {
    capturedURL = url;
    capturedOptions = options;
    return jsonResponse(201, null, { "content-length": "0" });
  };

  await addContact("bob");

  assert.equal(capturedURL, "http://localhost:8080/api/contacts");
  assert.equal(capturedOptions.method, "POST");
  assert.deepEqual(JSON.parse(capturedOptions.body), { username: "bob" });
});

test("removeContact deletes by contact id", async () => {
  process.env.VITE_API_BASE_URL = "http://localhost:8080";
  setToken("token-remove");

  let capturedOptions = null;
  global.fetch = async (_url, options) => {
    capturedOptions = options;
    return jsonResponse(200, null, { "content-length": "0" });
  };

  await removeContact(42);

  assert.equal(capturedOptions.method, "DELETE");
  assert.deepEqual(JSON.parse(capturedOptions.body), { contact_id: 42 });
});

test("getInbox sends query params for thread and unread filters", async () => {
  process.env.VITE_API_BASE_URL = "http://localhost:8080";
  setToken("token-inbox");

  let capturedURL = "";
  global.fetch = async (url) => {
    capturedURL = url;
    return jsonResponse(200, []);
  };

  await getInbox({ withUserID: 7, unreadOnly: true, limit: 25 });

  assert.equal(capturedURL, "http://localhost:8080/api/messages/inbox?limit=25&with_user_id=7&unread_only=true");
});

test("getOutbox sends pagination query params", async () => {
  process.env.VITE_API_BASE_URL = "http://localhost:8080";
  setToken("token-outbox");

  let capturedURL = "";
  global.fetch = async (url) => {
    capturedURL = url;
    return jsonResponse(200, []);
  };

  await getOutbox({ afterID: 9, limit: 10 });

  assert.equal(capturedURL, "http://localhost:8080/api/messages/outbox?limit=10&after_id=9");
});

test("getMessageThreads sends limit query params to messaging threads endpoint", async () => {
  process.env.VITE_API_BASE_URL = "http://localhost:8080";
  setToken("token-threads");

  let capturedURL = "";
  global.fetch = async (url) => {
    capturedURL = url;
    return jsonResponse(200, []);
  };

  await getMessageThreads({ limit: 30 });

  assert.equal(capturedURL, "http://localhost:8080/api/messaging/threads?limit=30");
});

test("syncMessages sends cursor query params to messaging sync endpoint", async () => {
  process.env.VITE_API_BASE_URL = "http://localhost:8080";
  setToken("token-sync");

  let capturedURL = "";
  global.fetch = async (url) => {
    capturedURL = url;
    return jsonResponse(200, { messages: [] });
  };

  await syncMessages({ afterID: 12, limit: 20 });

  assert.equal(capturedURL, "http://localhost:8080/api/messaging/sync?limit=20&after_id=12");
});

test("markMessageRead posts message id", async () => {
  process.env.VITE_API_BASE_URL = "http://localhost:8080";
  setToken("token-read");

  let capturedOptions = null;
  global.fetch = async (_url, options) => {
    capturedOptions = options;
    return jsonResponse(200, null, { "content-length": "0" });
  };

  await markMessageRead(52);

  assert.equal(capturedOptions.method, "POST");
  assert.deepEqual(JSON.parse(capturedOptions.body), { message_id: 52 });
});

test("markMessageDelivered posts message id", async () => {
  process.env.VITE_API_BASE_URL = "http://localhost:8080";
  setToken("token-delivered");

  let capturedOptions = null;
  global.fetch = async (_url, options) => {
    capturedOptions = options;
    return jsonResponse(200, null, { "content-length": "0" });
  };

  await markMessageDelivered(53);

  assert.equal(capturedOptions.method, "POST");
  assert.deepEqual(JSON.parse(capturedOptions.body), { message_id: 53 });
});

test("updateMe patches mutable profile fields", async () => {
  process.env.VITE_API_BASE_URL = "http://localhost:8080";
  setToken("token-profile");

  let capturedURL = "";
  let capturedOptions = null;
  global.fetch = async (url, options) => {
    capturedURL = url;
    capturedOptions = options;
    return jsonResponse(200, { username: "alice" });
  };

  await updateMe({ display_name: "Alice Doe", avatar_url: "https://example.com/alice.png" });

  assert.equal(capturedURL, "http://localhost:8080/api/me");
  assert.equal(capturedOptions.method, "PATCH");
  assert.deepEqual(JSON.parse(capturedOptions.body), {
    display_name: "Alice Doe",
    avatar_url: "https://example.com/alice.png",
  });
});

test("getWalletTransfers sends history limit query params", async () => {
  process.env.VITE_API_BASE_URL = "http://localhost:8080";
  setToken("token-history");

  let capturedURL = "";
  global.fetch = async (url) => {
    capturedURL = url;
    return jsonResponse(200, []);
  };

  await getWalletTransfers({ limit: 5 });

  assert.equal(capturedURL, "http://localhost:8080/api/wallet/transfers?limit=5");
});

test("invalid token response triggers auth error handler", async () => {
  process.env.VITE_API_BASE_URL = "http://localhost:8080";
  setToken("expired");

  let receivedError = "";
  setAuthErrorHandler((err) => {
    receivedError = err;
  });

  global.fetch = async () => jsonResponse(401, { error: "invalid token" });

  await assert.rejects(() => getContacts(), /invalid token/);
  assert.equal(receivedError, "invalid token");

  setAuthErrorHandler(null);
});
