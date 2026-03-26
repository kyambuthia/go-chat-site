import assert from "node:assert/strict";
import test from "node:test";

import {
  addContact,
  clearAuthState,
  ensureAccessToken,
  getAccessToken,
  getContacts,
  getDeviceDirectory,
  getDevices,
  getInbox,
  getMessageThreads,
  getOutbox,
  getSessions,
  getWalletTransfers,
  hasStoredSession,
  loginUser,
  logoutSession,
  markMessageDelivered,
  markMessageRead,
  markThreadRead,
  publishDevicePrekeys,
  removeContact,
  registerDeviceIdentity,
  revokeDeviceIdentity,
  revokeSession,
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
        const key = Object.keys(headers).find((headerName) => headerName.toLowerCase() === String(name).toLowerCase());
        return key ? headers[key] : null;
      },
    },
    async json() {
      return payload;
    },
  };
}

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
    clear() {
      values.clear();
    },
  };
}

test.beforeEach(() => {
  process.env.VITE_API_BASE_URL = "http://localhost:8080";
  process.env.VITE_WS_URL = "";
  global.localStorage = createStorage();
  clearAuthState();
  setAuthErrorHandler(null);
});

test.afterEach(() => {
  clearAuthState();
  setAuthErrorHandler(null);
  delete global.fetch;
  delete global.localStorage;
  delete process.env.VITE_API_BASE_URL;
  delete process.env.VITE_WS_URL;
});

test("loginUser stores access and refresh tokens", async () => {
  global.fetch = async (_url, options) => {
    assert.equal(options.method, "POST");
    assert.deepEqual(JSON.parse(options.body), {
      username: "alice",
      password: "password123",
      device_label: "MacBook Air",
    });
    return jsonResponse(200, {
      token: "access-1",
      access_token: "access-1",
      refresh_token: "refresh-1",
      access_token_expires_at: "2099-01-01T00:00:00Z",
      refresh_token_expires_at: "2099-02-01T00:00:00Z",
      session: { id: 17, device_label: "MacBook Air" },
    });
  };

  const response = await loginUser("alice", "password123", "MacBook Air");

  assert.equal(response.access_token, "access-1");
  assert.equal(getAccessToken(), "access-1");
  assert.equal(hasStoredSession(), true);
  assert.equal(global.localStorage.getItem("token"), "access-1");
  const stored = JSON.parse(global.localStorage.getItem("auth_session"));
  assert.equal(stored.refreshToken, "refresh-1");
  assert.equal(stored.session.id, 17);
});

test("ensureAccessToken refreshes an expired access token", async () => {
  let refreshCalls = 0;
  global.fetch = async (url, options) => {
    if (url === "http://localhost:8080/api/login") {
      return jsonResponse(200, {
        token: "access-old",
        access_token: "access-old",
        refresh_token: "refresh-1",
        access_token_expires_at: "2000-01-01T00:00:00Z",
        refresh_token_expires_at: "2099-02-01T00:00:00Z",
        session: { id: 5, device_label: "This browser" },
      });
    }
    if (url === "http://localhost:8080/api/auth/refresh") {
      refreshCalls += 1;
      assert.equal(options.method, "POST");
      assert.deepEqual(JSON.parse(options.body), {
        refresh_token: "refresh-1",
        device_label: "This browser",
      });
      return jsonResponse(200, {
        token: "access-new",
        access_token: "access-new",
        refresh_token: "refresh-2",
        access_token_expires_at: "2099-01-01T00:00:00Z",
        refresh_token_expires_at: "2099-03-01T00:00:00Z",
        session: { id: 5, device_label: "This browser" },
      });
    }
    throw new Error(`unexpected fetch ${url}`);
  };

  await loginUser("alice", "password123");
  const token = await ensureAccessToken("This browser");

  assert.equal(token, "access-new");
  assert.equal(getAccessToken(), "access-new");
  assert.equal(refreshCalls, 1);
});

test("invalid token response refreshes once and retries the request", async () => {
  const calls = [];
  global.fetch = async (url, options = {}) => {
    calls.push({ url, options });
    if (url === "http://localhost:8080/api/login") {
      return jsonResponse(200, {
        token: "access-1",
        access_token: "access-1",
        refresh_token: "refresh-1",
        access_token_expires_at: "2099-01-01T00:00:00Z",
        refresh_token_expires_at: "2099-02-01T00:00:00Z",
        session: { id: 9, device_label: "Desk" },
      });
    }
    if (url === "http://localhost:8080/api/contacts" && calls.filter((call) => call.url.endsWith("/api/contacts")).length === 1) {
      assert.equal(options.headers.Authorization, "Bearer access-1");
      return jsonResponse(401, { error: "invalid token" });
    }
    if (url === "http://localhost:8080/api/auth/refresh") {
      assert.equal(options.headers.Authorization, undefined);
      return jsonResponse(200, {
        token: "access-2",
        access_token: "access-2",
        refresh_token: "refresh-2",
        access_token_expires_at: "2099-01-02T00:00:00Z",
        refresh_token_expires_at: "2099-02-02T00:00:00Z",
        session: { id: 9, device_label: "Desk" },
      });
    }
    if (url === "http://localhost:8080/api/contacts") {
      assert.equal(options.headers.Authorization, "Bearer access-2");
      return jsonResponse(200, []);
    }
    throw new Error(`unexpected fetch ${url}`);
  };

  await loginUser("alice", "password123");
  await getContacts();

  assert.equal(getAccessToken(), "access-2");
  assert.equal(calls.filter((call) => call.url.endsWith("/api/auth/refresh")).length, 1);
});

test("logoutSession posts to logout endpoint", async () => {
  setToken("token-logout");

  let capturedURL = "";
  let capturedOptions = null;
  global.fetch = async (url, options) => {
    capturedURL = url;
    capturedOptions = options;
    return jsonResponse(204, null, { "content-length": "0" });
  };

  await logoutSession();

  assert.equal(capturedURL, "http://localhost:8080/api/logout");
  assert.equal(capturedOptions.method, "POST");
  assert.equal(capturedOptions.headers.Authorization, "Bearer token-logout");
});

test("getSessions uses the protected sessions endpoint", async () => {
  setToken("token-sessions");

  let capturedURL = "";
  global.fetch = async (url, options) => {
    capturedURL = url;
    assert.equal(options.headers.Authorization, "Bearer token-sessions");
    return jsonResponse(200, [{ id: 1, current: true }]);
  };

  const sessions = await getSessions();

  assert.equal(capturedURL, "http://localhost:8080/api/sessions");
  assert.equal(sessions.length, 1);
});

test("getDevices uses the protected devices endpoint", async () => {
  setToken("token-devices");

  let capturedURL = "";
  global.fetch = async (url, options) => {
    capturedURL = url;
    assert.equal(options.headers.Authorization, "Bearer token-devices");
    return jsonResponse(200, [{ id: 3, label: "Laptop" }]);
  };

  const devices = await getDevices();

  assert.equal(capturedURL, "http://localhost:8080/api/devices");
  assert.equal(devices.length, 1);
  assert.equal(devices[0].label, "Laptop");
});

test("getDeviceDirectory resolves a contact device directory by username", async () => {
  setToken("token-directory");

  let capturedURL = "";
  global.fetch = async (url, options) => {
    capturedURL = url;
    assert.equal(options.headers.Authorization, "Bearer token-directory");
    return jsonResponse(200, { username: "alice", devices: [{ id: 7 }] });
  };

  const directory = await getDeviceDirectory("alice");

  assert.equal(capturedURL, "http://localhost:8080/api/devices/directory?username=alice");
  assert.equal(directory.devices[0].id, 7);
});

test("revokeSession deletes by session id", async () => {
  setToken("token-revoke");

  let capturedOptions = null;
  global.fetch = async (_url, options) => {
    capturedOptions = options;
    return jsonResponse(204, null, { "content-length": "0" });
  };

  await revokeSession(42);

  assert.equal(capturedOptions.method, "DELETE");
  assert.deepEqual(JSON.parse(capturedOptions.body), { session_id: 42 });
});

test("registerDeviceIdentity posts the device identity payload", async () => {
  setToken("token-device-register");

  let capturedOptions = null;
  global.fetch = async (_url, options) => {
    capturedOptions = options;
    return jsonResponse(201, { id: 5, label: "Phone" });
  };

  await registerDeviceIdentity({
    label: "Phone",
    algorithm: "x3dh-ed25519-x25519-v1",
    identity_key: "identity-public",
    signed_prekey_id: 77,
    signed_prekey: "signed-prekey-public",
    signed_prekey_signature: "signature",
    prekeys: [{ prekey_id: 1, public_key: "prekey-public" }],
  });

  assert.equal(capturedOptions.method, "POST");
  assert.deepEqual(JSON.parse(capturedOptions.body), {
    label: "Phone",
    algorithm: "x3dh-ed25519-x25519-v1",
    identity_key: "identity-public",
    signed_prekey_id: 77,
    signed_prekey: "signed-prekey-public",
    signed_prekey_signature: "signature",
    prekeys: [{ prekey_id: 1, public_key: "prekey-public" }],
  });
});

test("publishDevicePrekeys posts the device id and prekeys", async () => {
  setToken("token-device-prekeys");

  let capturedOptions = null;
  global.fetch = async (_url, options) => {
    capturedOptions = options;
    return jsonResponse(200, [{ prekey_id: 2 }]);
  };

  await publishDevicePrekeys(19, [{ prekey_id: 2, public_key: "prekey-two" }]);

  assert.equal(capturedOptions.method, "POST");
  assert.deepEqual(JSON.parse(capturedOptions.body), {
    device_id: 19,
    prekeys: [{ prekey_id: 2, public_key: "prekey-two" }],
  });
});

test("revokeDeviceIdentity deletes by device id", async () => {
  setToken("token-device-revoke");

  let capturedOptions = null;
  global.fetch = async (_url, options) => {
    capturedOptions = options;
    return jsonResponse(204, null, { "content-length": "0" });
  };

  await revokeDeviceIdentity(55);

  assert.equal(capturedOptions.method, "DELETE");
  assert.deepEqual(JSON.parse(capturedOptions.body), { device_id: 55 });
});

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

test("markThreadRead posts counterparty user id", async () => {
  setToken("token-read-thread");

  let capturedURL = "";
  let capturedOptions = null;
  global.fetch = async (url, options) => {
    capturedURL = url;
    capturedOptions = options;
    return jsonResponse(200, null, { "content-length": "0" });
  };

  await markThreadRead(7);

  assert.equal(capturedURL, "http://localhost:8080/api/messaging/read-thread");
  assert.equal(capturedOptions.method, "POST");
  assert.deepEqual(JSON.parse(capturedOptions.body), { with_user_id: 7 });
});

test("markMessageDelivered posts message id", async () => {
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
  setToken("token-history");

  let capturedURL = "";
  global.fetch = async (url) => {
    capturedURL = url;
    return jsonResponse(200, []);
  };

  await getWalletTransfers({ limit: 5 });

  assert.equal(capturedURL, "http://localhost:8080/api/wallet/transfers?limit=5");
});

test("invalid token response without refresh token triggers auth error handler", async () => {
  setToken("expired");

  let receivedError = "";
  setAuthErrorHandler((err) => {
    receivedError = err;
  });

  global.fetch = async () => jsonResponse(401, { error: "invalid token" });

  await assert.rejects(() => getContacts(), /invalid token/);
  assert.equal(receivedError, "invalid token");
});
