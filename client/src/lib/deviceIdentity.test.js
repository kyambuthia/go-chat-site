import assert from "node:assert/strict";
import test from "node:test";

import {
  DEFAULT_DEVICE_ALGORITHM,
  DEFAULT_ENVELOPE_VERSION,
  buildOpaqueEnvelopeScaffold,
  formatPrekeysForTextarea,
  generateDeviceIdentityBundle,
  selectPreferredActiveDevice,
  selectPreferredRecipientDevice,
} from "./deviceIdentity.js";

test("generateDeviceIdentityBundle creates local key material and prekeys", () => {
  const bundle = generateDeviceIdentityBundle({ prekeyCount: 3 });

  assert.equal(bundle.algorithm, DEFAULT_DEVICE_ALGORITHM);
  assert.equal(typeof bundle.identity_key, "string");
  assert.equal(typeof bundle.signed_prekey, "string");
  assert.equal(typeof bundle.signed_prekey_signature, "string");
  assert.equal(Number.isInteger(bundle.signed_prekey_id), true);
  assert.equal(bundle.prekeys.length, 3);
  assert.ok(bundle.prekeys.every((prekey) => Number.isInteger(prekey.prekey_id) && typeof prekey.public_key === "string"));
});

test("formatPrekeysForTextarea serializes one prekey per line", () => {
  const text = formatPrekeysForTextarea([
    { prekey_id: 1, public_key: "alpha" },
    { prekey_id: 2, public_key: "beta" },
  ]);

  assert.equal(text, "1:alpha\n2:beta");
});

test("selectPreferredActiveDevice prefers the current active device", () => {
  const device = selectPreferredActiveDevice([
    { id: 1, state: "active", current_session: false },
    { id: 2, state: "active", current_session: true },
  ]);

  assert.equal(device.id, 2);
});

test("selectPreferredRecipientDevice returns the first active directory device", () => {
  const device = selectPreferredRecipientDevice({
    devices: [
      { id: 10, state: "revoked" },
      { id: 11, state: "active" },
    ],
  });

  assert.equal(device.id, 11);
});

test("buildOpaqueEnvelopeScaffold returns optional envelope metadata", () => {
  const envelope = buildOpaqueEnvelopeScaffold({
    body: "hello",
    senderDeviceID: 7,
    recipientDeviceID: 9,
  });

  assert.equal(typeof envelope.ciphertext, "string");
  assert.equal(envelope.encryption_version, DEFAULT_ENVELOPE_VERSION);
  assert.equal(envelope.sender_device_id, 7);
  assert.equal(envelope.recipient_device_id, 9);
});
