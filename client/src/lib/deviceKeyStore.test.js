import assert from "node:assert/strict";
import test from "node:test";

import {
  getLocalDeviceBundle,
  hasLocalDeviceBundle,
  listLocalDeviceBundleIDs,
  removeLocalDeviceBundle,
  saveLocalDeviceBundle,
} from "./deviceKeyStore.js";

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
  global.localStorage = createStorage();
});

test.afterEach(() => {
  delete global.localStorage;
});

test("saveLocalDeviceBundle stores and retrieves a private bundle", () => {
  const bundle = { identity_private_key: "secret" };

  saveLocalDeviceBundle({ userID: 5, deviceID: 12, bundle });

  assert.equal(hasLocalDeviceBundle(12), true);
  assert.deepEqual(getLocalDeviceBundle(12), bundle);
  assert.deepEqual(listLocalDeviceBundleIDs(), [12]);
});

test("removeLocalDeviceBundle deletes stored device material", () => {
  saveLocalDeviceBundle({ userID: 8, deviceID: 99, bundle: { identity_private_key: "secret" } });

  removeLocalDeviceBundle(99);

  assert.equal(hasLocalDeviceBundle(99), false);
  assert.equal(getLocalDeviceBundle(99), null);
  assert.deepEqual(listLocalDeviceBundleIDs(), []);
});
