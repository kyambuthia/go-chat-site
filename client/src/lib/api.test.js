import assert from "node:assert/strict";
import test from "node:test";

import { getContacts, sendMoney, setAuthErrorHandler, setToken } from "../api.js";

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
